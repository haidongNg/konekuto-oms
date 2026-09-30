package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/jmoiron/sqlx"

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

// fiberServer là struct đóng gói
type fiberServer struct {
	app        *fiber.App
	db         *sqlx.DB
	cfg        *config.Config
	httpServer *http.Server
}

// NewServer khởi tạo và trả về Interface Server
func NewServer(cfg *config.Config, db *sqlx.DB) Server {
	app := fiber.New(
		fiber.Config{
			StructValidator: validations.NewValidator(),
			ErrorHandler:    customHTTPErrorHandler,
			BodyLimit:       2 * 1024 * 1024, // Giới hạn kích thước body request là 2MB
			// 1. Dùng bộ phân giải JSON tốc độ cao (Nhanh hơn 3-5 lần chuẩn Go)
			JSONEncoder: json.Marshal,
			JSONDecoder: json.Unmarshal,
			// 3. Tối ưu Header
			ServerHeader: "Fiber", // Trả về header server gọn nhẹ
			// 4. Các cấu hình Timeout (Chống tấn công DDoS ngâm kết nối Slowloris)
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
	)

	return &fiberServer{
		app: app,
		db:  db,
		cfg: cfg,
	}
}

func customHTTPErrorHandler(c fiber.Ctx, err error) error {
	code := http.StatusInternalServerError
	message := "Lỗi máy chủ nội bộ"
	if he, ok := err.(*fiber.Error); ok {
		code = he.Code
		message = fmt.Sprintf("%v", he.Message)
	}
	return response.Error(c, code, message)
}

func (s *fiberServer) Run() error {
	s.mapMiddlewares()
	s.mapHandlers()
	slog.Info("🚀 Server khởi động", "port", s.cfg.Server.Port)
	// Lưu ý: Fiber sẽ tự động bỏ qua Prefork nếu bạn chạy code trên Windows.
	return s.app.Listen(s.cfg.Server.Port, fiber.ListenConfig{
		EnablePrefork:         false, // Bật đa tiến trình để tối đa hoá RPS
		DisableStartupMessage: false, // Để false để xem logo và log port của Fiber lúc khởi động
	})
}

// Shutdown xử lý Graceful Shutdown
func (s *fiberServer) Shutdown(ctx context.Context) error {
	slog.Info("Dừng Fiber Server...")
	// Sử dụng ShutdownWithContext để Fiber tuân thủ thời gian timeout (10s) mà bạn đã set ở main
	return s.app.ShutdownWithContext(ctx)
}

func (s *fiberServer) mapMiddlewares() {
	// 3. ETag Middleware
	s.app.Use(etag.New(etag.Config{
		Weak: true, // Sử dụng ETag Weak để giảm bớt việc tính toán hash
	}))

	// 3. CORS middleware
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"https://127.0.0.1:8080", "https://localhost:8080", "http://localhost:8080"},
		AllowHeaders: []string{
			"Origin",
			fiber.HeaderContentType,
			"Accept",
			fiber.HeaderAuthorization,
			"X-Request-ID",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
	}))

	// 4. Rate Limiter middleware
	s.app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 1 * time.Minute,
	}))

	// 5. Request Logger custom với slog
	s.app.Use(func(c fiber.Ctx) error {
		start := time.Now()

		reqID := c.Get("X-Request-ID")

		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()

		logData := []slog.Attr{
			slog.String("req_id", reqID),
			slog.String("method", c.Method()),
			slog.String("uri", c.OriginalURL()),
			slog.Int("status", status),
			slog.Duration("latency", latency),
		}

		if err == nil {
			slog.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", logData...)
		} else {
			logData = append(logData, slog.String("err", err.Error()))
			slog.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR", logData...)
		}

		return err
	})
}

func (s *fiberServer) mapHandlers() {
	timeoutContext := s.cfg.Server.Timeout * time.Second
	jwtSecret := s.cfg.Security.JWTSecret

	// Lắp ráp Module User bằng Dependency Injection
	uRepo := userRepo.NewSQLiteUserRepository(s.db)
	uUseCase := userUseCase.NewUserUseCase(uRepo, timeoutContext, jwtSecret)
	uHandler := userDelivery.NewUserHandler(uUseCase)

	uHandler.RegisterRoutes(s.app, jwtSecret)
	// KÍCH HOẠT JOB DỌN RÁC NGẦM
	s.startCleanupTask(uUseCase)

	// =========================================
	// 2. Lắp ráp Module Sản Phẩm (MỚI THÊM)
	// =========================================
	pRepo := productRepo.NewSQLiteProductRepository(s.db)
	pUseCase := productUseCase.NewProductUseCase(pRepo, timeoutContext)
	pHandler := productDelivery.NewProductHandler(pUseCase)

	// Truyền uUseCase vào làm checker để kiểm tra Blacklist cho các API Admin
	pHandler.RegisterRoutes(s.app, jwtSecret, uUseCase)

	// =========================================
	// 3. Lắp ráp Module Đơn Hàng (Order)
	// =========================================
	oRepo := orderRepo.NewSQLiteOrderRepository(s.db)

	// CHÚ Ý: Tiêm cả oRepo (lưu Order) và pRepo (để UseCase dò giá Sản phẩm)
	oUseCase := orderUseCase.NewOrderUseCase(oRepo, pRepo, timeoutContext)

	oHandler := orderDelivery.NewOrderHandler(oUseCase)
	oHandler.RegisterRoutes(s.app, jwtSecret, uUseCase)
}

// startCleanupTask là một Background Job chạy ngầm để dọn dẹp database
func (s *fiberServer) startCleanupTask(uUseCase domain.UserUseCase) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Gọi thẳng xuống Tầng UseCase, Server không cần biết DB chạy lệnh gì
			_ = uUseCase.CleanupExpiredTokens(context.Background())
		}
	}()
}
