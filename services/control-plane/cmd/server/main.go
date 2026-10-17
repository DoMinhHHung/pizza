package main

import (
	"context"
	"log/slog"
	"net"
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
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("listen grpc", "error", err)
		os.Exit(1)
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
	reflection.Register(grpcServer)

	go func() {
		slog.Info("gRPC server starting", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
			stop()
		}
	}()

	httpServer := httpapi.New(validator, orgService, projectService, apiKeyService, pool.Ping)
	go func() {
		addr := ":" + cfg.HTTPPort
		slog.Info("HTTP API starting", "port", cfg.HTTPPort)
		if err := httpapi.ListenAndServe(addr, httpServer.Handler()); err != nil {
			slog.Error("HTTP server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	grpcServer.GracefulStop()
	slog.Info("server stopped")
}
