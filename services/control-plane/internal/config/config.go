package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTPPort string `env:"HTTP_PORT" envDefault:"8081"`
	GRPCPort string `env:"GRPC_PORT" envDefault:"9090"`

	DatabaseURL string `env:"DATABASE_URL,required"`

	JWTSecret string `env:"JWT_SECRET,required"`

	// reserved for future JWKS
	// JWKSURL string `env:"JWKS_URL"`

	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
