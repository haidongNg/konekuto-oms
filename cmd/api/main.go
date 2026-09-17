package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haidongNg/konekuto-oms/internal/config"
	"github.com/haidongNg/konekuto-oms/internal/server"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.LoadConfig("configs/config.local.yaml")
	if err != nil {
		os.Exit(1)
	}

	db, err := sqlx.Connect(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		slog.Error("❌ Không thể khởi tạo database", "error", err)
		os.Exit(1)
	}

	srv := server.NewServer(cfg, db)

	go func() {
		// Bỏ qua lỗi ErrServerClosed vì đó là lỗi bình thường khi ta chủ động gọi Shutdown
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("❌ Server crash", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("🔴 Đang tắt server an toàn...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// KHẮC PHỤC: Gọi thẳng hàm Shutdown của Server struct
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Lỗi Shutdown HTTP", "error", err)
	}
	if err := db.Close(); err != nil {
		slog.Error("Lỗi Shutdown DB", "error", err)
	}
	slog.Info("✅ Tắt hoàn tất.")
}
