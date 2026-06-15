package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"gitlab.praktikum-services.ru/Stasyan/momo-store/cmd/api/app"
	"gitlab.praktikum-services.ru/Stasyan/momo-store/cmd/api/dependencies"
	"gitlab.praktikum-services.ru/Stasyan/momo-store/internal/logger"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			os.Exit(1)
		}
		return
	}

	logger.Setup(os.Getenv("LOG_LEVEL"))

	if err := run(); err != nil {
		logger.Log.Fatal("unexpected error", zap.Error(err))
		os.Exit(1)
	}
}

func run() error {
	port := envOrDefault("APP_PORT", "8081")
	address := ":" + port

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	secretPath := envOrDefault("ORDER_ID_SECRET_FILE", "/run/secrets/order_id_secret")
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("cannot read order ID secret: %w", err)
	}
	secret = bytes.TrimSpace(secret)
	if len(secret) < 32 {
		return fmt.Errorf("order ID secret must contain at least 32 bytes")
	}

	orderDBPath := envOrDefault("ORDER_DB_PATH", "/data/orders.seq")
	store, err := dependencies.NewPersistentDumplingsStore(orderDBPath, secret)
	if err != nil {
		return fmt.Errorf("cannot bootstrap dumplings store: %w", err)
	}

	logger.Log.Debug("creating app instance")
	instance, err := app.NewInstance(store)
	if err != nil {
		return fmt.Errorf("cannot create app instance: %w", err)
	}

	router, err := newRouter(instance)
	if err != nil {
		return fmt.Errorf("cannot create router instance: %w", err)
	}

	srv := &http.Server{
		Handler: router,
	}

	errChan := make(chan error, 1)
	go func() {
		logger.Log.Info("starting HTTP server", zap.String("address", address))
		if err := srv.Serve(lis); err != nil {
			errChan <- fmt.Errorf("error serving HTTP: %w", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logger.Log.Info("shutting down gracefully", zap.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return srv.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

func healthcheck() error {
	port := envOrDefault("APP_PORT", "8081")
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected health status: %s", response.Status)
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
