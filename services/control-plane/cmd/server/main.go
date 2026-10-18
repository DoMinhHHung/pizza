package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	controlv1 "github.com/DoMinhHHung/pizza/services/control-plane/api/gen/control/v1"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/apikey"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/auth"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/config"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/db"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/httpapi"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/org"
	"github.com/DoMinhHHung/pizza/services/control-plane/internal/project"
)

func main() {
	if err := run(); err != nil {
		slog.Error("control-plane exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	addr := net.JoinHostPort(cfg.GRPCHost, cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	validator := auth.NewJWTValidator(cfg.JWTSecret)

	orgRepo := org.NewRepository(pool)
	orgService := org.NewService(orgRepo)
	projectRepo := project.NewRepository(pool)
	projectService := project.NewService(projectRepo, orgRepo)
	apiKeyRepo := apikey.NewRepository(pool)
	apiKeyService := apikey.NewService(apiKeyRepo, projectRepo, orgRepo)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(validator.UnaryInterceptor()),
	)
	controlv1.RegisterOrganizationServiceServer(grpcServer, orgService)
	controlv1.RegisterProjectServiceServer(grpcServer, projectService)
	controlv1.RegisterApiKeyServiceServer(grpcServer, apiKeyService)
	if cfg.GRPCReflectionEnabled {
		reflection.Register(grpcServer)
	}

	api := httpapi.New(validator, orgService, projectService, apiKeyService, pool.Ping)
	httpServer := httpapi.NewHTTPServer(":"+cfg.HTTPPort, api.Handler())

	serveErr := make(chan error, 2)
	go func() {
		slog.Info("gRPC server starting", "addr", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			serveErr <- fmt.Errorf("grpc serve: %w", err)
		}
	}()
	go func() {
		slog.Info("HTTP API starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("http serve: %w", err)
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case runErr = <-serveErr:
		slog.Error("server failed", "error", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	shutdown(shutdownCtx, grpcServer, httpServer)
	slog.Info("server stopped")
	return runErr
}

func shutdown(ctx context.Context, grpcServer *grpc.Server, httpServer *http.Server) {
	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("http shutdown", "error", err)
		_ = httpServer.Close()
	}

	select {
	case <-grpcDone:
	case <-ctx.Done():
		grpcServer.Stop()
		<-grpcDone
	}
}
