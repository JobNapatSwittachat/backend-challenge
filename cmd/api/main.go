// Command api starts the user management HTTP and gRPC servers.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"user-management-api/internal/adapter/auth"
	grpchandler "user-management-api/internal/adapter/handler/grpc"
	userv1 "user-management-api/internal/adapter/handler/grpc/gen/user/v1"
	httphandler "user-management-api/internal/adapter/handler/http"
	mongorepo "user-management-api/internal/adapter/repository/mongo"
	"user-management-api/internal/config"
	"user-management-api/internal/core/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Root context is cancelled on SIGINT/SIGTERM to trigger graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Infrastructure (driven adapters) ---
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongorepo.Connect(connectCtx, cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			logger.Error("disconnecting mongodb", "error", err)
		}
	}()
	logger.Info("connected to mongodb")

	repo, err := mongorepo.NewUserRepository(connectCtx, client.Database(cfg.MongoDB))
	if err != nil {
		return err
	}
	hasher := auth.NewBcryptHasher()
	tokens := auth.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)

	// --- Core ---
	userService := service.NewUserService(repo, hasher, tokens)

	// --- Background concurrency task: log user count every 10s ---
	var wg sync.WaitGroup
	counter := service.NewUserCounter(userService, logger, cfg.CountEvery)
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter.Run(ctx)
	}()

	// --- HTTP server (driving adapter) ---
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httphandler.NewRouter(userService, tokens, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 2)
	go func() {
		logger.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// --- gRPC server (driving adapter) ---
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpchandler.AuthUnaryInterceptor(tokens)))
	userv1.RegisterUserServiceServer(grpcServer, grpchandler.NewServer(userService))
	// Reflection lets tools like grpcurl discover the API without the proto file.
	reflection.Register(grpcServer)
	go func() {
		listener, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			errCh <- err
			return
		}
		logger.Info("grpc server listening", "addr", cfg.GRPCAddr)
		if err := grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	// Block until a shutdown signal or a server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	// --- Graceful shutdown ---
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownWait)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown", "error", err)
	}
	grpcServer.GracefulStop()
	wg.Wait()

	logger.Info("shutdown complete")
	return nil
}
