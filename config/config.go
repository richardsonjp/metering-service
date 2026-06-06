package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	System   System
	Metering Metering
	Log      Logging
}

type System struct {
	AppName  string `env:"SYSTEM_APP_NAME"  envDefault:"metering-service"`
	Addr     string `env:"SERVER_ADDR"      envDefault:":8080"`
	TimeZone string `env:"SYSTEM_TIME_ZONE" envDefault:"Asia/Jakarta"`
}

type Metering struct {
	RequestLimit   int64 `env:"REQUEST_LIMIT"        envDefault:"1000"`
	StorageLimit   int64 `env:"STORAGE_LIMIT_BYTES"  envDefault:"1073741824"`
	MaxUploadBytes int64 `env:"MAX_UPLOAD_BYTES"     envDefault:"1073741824"`
}

type Logging struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() Config {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			panic(fmt.Errorf("config: loading .env: %w", err))
		}
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		panic(fmt.Errorf("config: parsing environment: %w", err))
	}
	return cfg
}

func (c Config) Show() {
	out, _ := json.MarshalIndent(c, "", "  ")
	fmt.Printf("Config: %s\n", out)
}
