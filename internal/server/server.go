package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/haidongNg/konekuto-oms/internal/config"
	"github.com/haidongNg/konekuto-oms/internal/domain"
	orderDelivery "github.com/haidongNg/konekuto-oms/internal/order/delivery/http"
	orderRepo "github.com/haidongNg/konekuto-oms/internal/order/repository"
	orderUseCase "github.com/haidongNg/konekuto-oms/internal/order/usecase"
	productDelivery "github.com/haidongNg/konekuto-oms/internal/product/delivery/http"
	productRepo "github.com/haidongNg/konekuto-oms/internal/product/repository"
	productUseCase "github.com/haidongNg/konekuto-oms/internal/product/usecase"
	userDelivery "github.com/haidongNg/konekuto-oms/internal/user/delivery/http"
	userRepo "github.com/haidongNg/konekuto-oms/internal/user/repository"
	userUseCase "github.com/haidongNg/konekuto-oms/internal/user/usecase"
	"github.com/haidongNg/konekuto-oms/pkg/response"
	"github.com/haidongNg/konekuto-oms/pkg/validations"
)

// Server định nghĩa vòng đời của HTTP Server
type Server interface {
	Run() error
	Shutdown(ctx context.Context) error
}

// echoServer là struct đóng gói
type echoServer struct {
	echo       *echo.Echo
	db         *sqlx.DB
	cfg        *config.Config
	httpServer *http.Server
}

// NewServer khởi tạo và trả về Interface Server
func NewServer(cfg *config.Config, db *sqlx.DB) Server {
	e := echo.New()
	e.Validator = validations.NewValidator()
	e.HTTPErrorHandler = customHTTPErrorHandler

	return &echoServer{
		echo: e,
		db:   db,
		cfg:  cfg,
	}
}

func customHTTPErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	message := "Lỗi máy chủ nội bộ"
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = fmt.Sprintf("%v", he.Message)
	}
	_ = response.Error(c, code, message)
}

func (s *echoServer) Run() error {
	s.mapMiddlewares()
	s.mapHandlers()

	s.httpServer = &http.Server{
		Addr:    s.cfg.Server.Port,
		Handler: s.echo, // Sử dụng nguyên lý net/http chuẩn
	}

	slog.Info("🚀 Server khởi động", "port", s.cfg.Server.Port)
	return s.httpServer.ListenAndServe()
}

func (s *echoServer) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *echoServer) mapMiddlewares() {
	s.echo.Use(middleware.Recover())
	s.echo.Use(middleware.RequestID())
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://127.0.0.1:8080", "https://localhost:8080", "http://localhost:8080"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
	}))
	s.echo.Use(middleware.Secure())
	s.echo.Use(middleware.BodyLimit(2_097_152)) // Giới hạn kích thước body request là 2MB
	s.echo.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	s.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true, LogURI: true, LogMethod: true, LogLatency: true, HandleError: true, LogHeaders: []string{echo.HeaderXRequestID},
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			reqID := ""
			if len(v.Headers[echo.HeaderXRequestID]) > 0 {
				reqID = v.Headers[echo.HeaderXRequestID][0]
			}
			logData := []slog.Attr{
				slog.String("req_id", reqID), slog.String("method", v.Method),
				slog.String("uri", v.URI), slog.Int("status", v.Status), slog.Duration("latency", v.Latency),
			}
			if v.Error == nil {
				slog.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", logData...)
			} else {
				logData = append(logData, slog.String("err", v.Error.Error()))
				slog.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR", logData...)
			}
			return nil
		},
	}))
}

func (s *echoServer) mapHandlers() {
	timeoutContext := s.cfg.Server.Timeout * time.Second
	jwtSecret := s.cfg.Security.JWTSecret

	// Lắp ráp Module User bằng Dependency Injection
	uRepo := userRepo.NewSQLiteUserRepository(s.db)
	uUseCase := userUseCase.NewUserUseCase(uRepo, timeoutContext, jwtSecret)
	uHandler := userDelivery.NewUserHandler(uUseCase)

	uHandler.RegisterRoutes(s.echo, jwtSecret)
	// KÍCH HOẠT JOB DỌN RÁC NGẦM
	s.startCleanupTask(uUseCase)

	// =========================================
	// 2. Lắp ráp Module Sản Phẩm (MỚI THÊM)
	// =========================================
	pRepo := productRepo.NewSQLiteProductRepository(s.db)
	pUseCase := productUseCase.NewProductUseCase(pRepo, timeoutContext)
	pHandler := productDelivery.NewProductHandler(pUseCase)

	// Truyền uUseCase vào làm checker để kiểm tra Blacklist cho các API Admin
	pHandler.RegisterRoutes(s.echo, jwtSecret, uUseCase)

	// =========================================
	// 3. Lắp ráp Module Đơn Hàng (Order)
	// =========================================
	oRepo := orderRepo.NewSQLiteOrderRepository(s.db)

	// CHÚ Ý: Tiêm cả oRepo (lưu Order) và pRepo (để UseCase dò giá Sản phẩm)
	oUseCase := orderUseCase.NewOrderUseCase(oRepo, pRepo, timeoutContext)

	oHandler := orderDelivery.NewOrderHandler(oUseCase)
	oHandler.RegisterRoutes(s.echo, jwtSecret, uUseCase)
}

// startCleanupTask là một Background Job chạy ngầm để dọn dẹp database
func (s *echoServer) startCleanupTask(uUseCase domain.UserUseCase) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Gọi thẳng xuống Tầng UseCase, Server không cần biết DB chạy lệnh gì
			_ = uUseCase.CleanupExpiredTokens(context.Background())
		}
	}()
}
