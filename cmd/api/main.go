package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"leadflow/internal/handler"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/repository"
	"leadflow/internal/service"
)

type config struct {
	HTTPAddr string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisAddr string
}

func loadConfig() config {
	return config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "leadflow"),
		DBPassword: getEnv("DB_Password", "leadflow"),
		DBName:     getEnv("DB_NAME", "leadflow"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := loadConfig()

	ctx := context.Background()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close(ctx)
	}()

	redisQueue := queue.NewRedisQueue(cfg.RedisAddr, "leads")

	leadRepo := repository.NewLeadRepo(db.Pool())
	orgRepo := repository.NewOrganizationRepo(db.Pool())
	webhookRepo := repository.NewWebhookRepo(db.Pool())
	apiKeyRepo := repository.NewAPIKeyRepo(db.Pool())

	leadSvc := service.NewLeadService(leadRepo, redisQueue)
	orgSvc := service.NewOrganizationService(orgRepo)
	webhookSvc := service.NewWebhookService(webhookRepo)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)

	leadHandler := handler.NewLeadHandler(leadSvc)
	leadListHandler := handler.NewLeadListHandler(leadSvc)
	orgHandler := handler.NewOrganizationHandler(orgSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)

	adminLeadsHandler, err := handler.NewAdminLeadsHandler(leadSvc)
	if err != nil {
		logger.Error("failed to create admin leads handler", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("POST /api/v1/leads", leadHandler.Create)
	mux.Handle("GET /api/v1/leads", leadListHandler)
	mux.Handle("GET /admin/leads", adminLeadsHandler)

	mux.HandleFunc("POST /api/v1/organizations", orgHandler.Create)

	mux.HandleFunc("POST /api/v1/webhooks", webhookHandler.Create)
	mux.HandleFunc("GET /api/v1/webhooks", webhookHandler.List)
	mux.HandleFunc("DELETE /api/v1/webhooks/", webhookHandler.Delete)

	mux.HandleFunc("POST /api/v1/api-keys", apiKeyHandler.Create)
	mux.HandleFunc("GET /api/v1/api-keys", apiKeyHandler.List)
	mux.HandleFunc("DELETE /api/v1/api-keys", apiKeyHandler.Delete)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           loggingMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("api server started", "address", server.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		logger.Error("api server failed", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)

		logger.Info("api server stopped")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()

		next.ServeHTTP(w, r)

		logger.Info(
			"http request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(startedAt).String(),
		)
	})
}
