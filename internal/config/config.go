package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type HTTPServer struct {
	Timeout      time.Duration `mapstructure:"timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
}

// ВАЖНО: поля названы в точности как в main.go:
// StoragePath, Address, HTTPServer.
type Config struct {
	Env         string     `mapstructure:"env"`
	Version     string     `mapstructure:"version"`
	Address     string     `mapstructure:"address"`
	StoragePath string     `mapstructure:"storage_path"`
	HTTPServer  HTTPServer `mapstructure:"http_server"`

	// Оставил лог-уровень, если он используется где-то ещё
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("local")
	v.SetConfigType("yaml")

	// Вариант №3 — несколько путей поиска
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("..")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")

	// Папки относительно бинарника (на случай go build)
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		v.AddConfigPath(base)
		v.AddConfigPath(filepath.Join(base, "config"))
		v.AddConfigPath(filepath.Join(base, ".."))
		v.AddConfigPath(filepath.Join(base, "..", "config"))
	}

	// Явный путь через переменную окружения (опционально)
	if p := os.Getenv("URL_SHORTENER_CONFIG"); p != "" {
		v.SetConfigFile(filepath.Clean(p))
	}

	// Значения по умолчанию (строки с суффиксом s/m/h парсятся как time.Duration)
	v.SetDefault("env", "local")
	v.SetDefault("version", "dev")
	v.SetDefault("address", "127.0.0.1:8080")
	v.SetDefault("storage_path", "storage/sqlite/urls.db")
	v.SetDefault("http_server.timeout", "4s")
	v.SetDefault("http_server.read_timeout", "4s")
	v.SetDefault("http_server.write_timeout", "4s")
	v.SetDefault("http_server.idle_timeout", "60s")
	v.SetDefault("log.level", "debug")

	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if errors.As(err, &nf) {
			return nil, fmt.Errorf("config file does not exist: tried multiple locations (e.g. ./config/local.yaml): %w", err)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
