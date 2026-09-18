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
		AllowOrigins: []string{"*"},
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
}
