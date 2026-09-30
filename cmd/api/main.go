package main

import (
	"context"
	"log/slog"
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
	// 1. TỐI ƯU LOGGER: Dùng JSON Handler để dễ dàng tích hợp với ElasticSearch, Kibana, Datadog...
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. LOAD CONFIG
	cfg, err := config.LoadConfig("configs/config.local.yaml")
	if err != nil {
		slog.Error("❌ Lỗi load cấu hình", "error", err)
		os.Exit(1)
	}

	// 3. KHỞI TẠO & TỐI ƯU DATABASE CONNECTION POOL
	db, err := sqlx.Connect(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		slog.Error("❌ Không thể kết nối Database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("⚠️ Lỗi đóng kết nối DB", "error", err)
		} else {
			slog.Info("✅ Kết nối DB đã được đóng an toàn")
		}
	}()

	// 🔥 Tối ưu Pool Kết Nối DB (Cực kỳ quan trọng để không nghẽn cổ chai)
	db.SetMaxOpenConns(100)                 // Số kết nối tối đa mở đồng thời
	db.SetMaxIdleConns(20)                  // Số kết nối nhàn rỗi giữ lại trong pool
	db.SetConnMaxLifetime(15 * time.Minute) // Tuổi thọ tối đa của 1 kết nối (tránh lỗi ngắt mạng ngầm)
	db.SetConnMaxIdleTime(5 * time.Minute)  // Thời gian tối đa giữ kết nối nhàn rỗi

	// 4. KHỞI TẠO FIBER SERVER
	srv := server.NewServer(cfg, db)

	// 5. CHẠY SERVER BẤT ĐỒNG BỘ
	go func() {
		// Ở Fiber, nếu chạy Run() trả về lỗi khác nil thì mới là crash
		if err := srv.Run(); err != nil {
			slog.Error("❌ Server Fiber bị crash", "error", err)
			os.Exit(1)
		}
	}()

	// 6. GRACEFUL SHUTDOWN (Tắt ứng dụng an toàn không làm rớt request)
	quit := make(chan os.Signal, 1)
	// Lắng nghe tín hiệu từ OS (Ctrl+C trên terminal, hoặc lệnh stop của Docker/Kubernetes)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit // Ứng dụng sẽ block ở đây cho đến khi nhận được tín hiệu

	slog.Info("🔴 Nhận tín hiệu tắt, đang dọn dẹp tài nguyên (Graceful Shutdown)...")

	// Cho phép Fiber tối đa 10 giây để xử lý xong các request đang dang dở trước khi ép tắt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("⚠️ Lỗi khi tắt HTTP Server", "error", err)
	} else {
		slog.Info("✅ HTTP Server đã tắt hoàn toàn")
	}
}
