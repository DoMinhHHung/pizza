package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

const minJWTSecretLength = 32

type Config struct {
	HTTPPort string `env:"HTTP_PORT" envDefault:"8081"`
	GRPCPort string `env:"GRPC_PORT" envDefault:"9090"`
	GRPCHost string `env:"GRPC_HOST"`

	GRPCReflectionEnabled bool `env:"GRPC_REFLECTION_ENABLED" envDefault:"false"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	JWTSecret   string `env:"JWT_SECRET,required"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`

	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}
	return &cfg, nil
}
