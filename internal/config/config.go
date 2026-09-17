package config

import (
	"log/slog"
	"time"

	"github.com/spf13/viper"
)

// Config chứa toàn bộ cấu hình hệ thống map từ file yaml
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port    string
	Timeout time.Duration
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type SecurityConfig struct {
	JWTSecret      string `mapstructure:"jwt_secret"`       // Cần tag mapstructure để Viper hiểu dấu gạch dưới
	JWTExpireHours int    `mapstructure:"jwt_expire_hours"` // Cần tag mapstructure
}

// LoadConfig đọc file yaml và nạp vào struct Config
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv() // Hỗ trợ đọc ghi đè từ biến môi trường (Environment Variables)

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("❌ Lỗi đọc file config", "error", err)
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		slog.Error("❌ Lỗi parse config", "error", err)
		return nil, err
	}

	return &config, nil
}
