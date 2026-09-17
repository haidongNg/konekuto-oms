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
	userDelivery "github.com/haidongNg/konekuto-oms/internal/user/delivery/http"
	userRepo "github.com/haidongNg/konekuto-oms/internal/user/repository"
	userUseCase "github.com/haidongNg/konekuto-oms/internal/user/usecase"

	"github.com/haidongNg/konekuto-oms/pkg/response"
	"github.com/haidongNg/konekuto-oms/pkg/validations"
)

// Server quản lý vòng đời HTTP Server và các dependency
type Server struct {
	echo       *echo.Echo
	db         *sqlx.DB
	cfg        *config.Config
	httpServer *http.Server // Quản lý máy chủ HTTP nguyên bản của Go
}

// NewServer khởi tạo instance Server
func NewServer(cfg *config.Config, db *sqlx.DB) *Server {
	e := echo.New()

	e.Validator = validations.NewValidator()
	e.HTTPErrorHandler = customHTTPErrorHandler

	return &Server{
		echo: e,
		db:   db,
		cfg:  cfg,
	}
}

// customHTTPErrorHandler bắt mọi lỗi ngoại lệ
func customHTTPErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	message := "Lỗi máy chủ nội bộ"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = fmt.Sprintf("%v", he.Message)
	}

	_ = response.Error(c, code, message)
}

// Run khởi động server bằng net/http chuẩn
func (s *Server) Run() error {
	s.mapMiddlewares()
	s.mapHandlers()

	s.httpServer = &http.Server{
		Addr:    s.cfg.Server.Port,
		Handler: s.echo,
	}

	slog.Info("🚀 Server khởi động", "port", s.cfg.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Shutdown giúp main.go tắt server an toàn
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// mapMiddlewares đăng ký toàn bộ Core Middleware
func (s *Server) mapMiddlewares() {
	s.echo.Use(middleware.Recover())
	s.echo.Use(middleware.RequestID())

	// CẤU HÌNH CORS TỐI ƯU (Khắc phục lỗi panic của Echo v5)
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Môi trường Dev cho phép tất cả, lên Prod cần trỏ đúng domain
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization, // Cần thiết để nhận token JWT
			echo.HeaderXRequestID,    // Cho phép client gửi kèm RequestID để trace log
		},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
	}))

	s.echo.Use(middleware.Secure())
	s.echo.Use(middleware.BodyLimit(2_097_152)) // Giới hạn 2MB cho body request
	s.echo.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	s.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		HandleError: true,
		LogHeaders:  []string{echo.HeaderXRequestID},
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			reqID := ""
			if len(v.Headers[echo.HeaderXRequestID]) > 0 {
				reqID = v.Headers[echo.HeaderXRequestID][0]
			}

			logData := []slog.Attr{
				slog.String("req_id", reqID),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
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

// mapHandlers đóng vai trò lắp ráp các module
func (s *Server) mapHandlers() {
	timeoutContext := s.cfg.Server.Timeout * time.Second
	jwtSecret := s.cfg.Security.JWTSecret

	// Lắp ráp Module User
	uRepo := userRepo.NewSQLiteUserRepository(s.db)
	uUseCase := userUseCase.NewUserUseCase(uRepo, timeoutContext, jwtSecret)

	userDelivery.NewUserHandler(s.echo, uUseCase, jwtSecret)
}
