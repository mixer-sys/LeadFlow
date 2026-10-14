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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"leadflow/internal/auth"
	"leadflow/internal/config"
	"leadflow/internal/handler"
	"leadflow/internal/logging"
	"leadflow/internal/middleware"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/repository"
	"leadflow/internal/service"
)

func main() {
	cfg := config.New()

	logger := logging.New(logging.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	slog.SetDefault(logger)

	logger.Info("starting leadflow",
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("log_level", cfg.LogLevel),
		slog.String("log_format", cfg.LogFormat),
	)

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

	webhookAdminHandler := handler.NewWebhookAdminHandler(webhookSvc)

	adminLeadsHandler, err := handler.NewAdminLeadsHandler(leadSvc)
	if err != nil {
		logger.Error("failed to create admin leads handler", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	apiKeyAuth := auth.APIKeyAuth(apiKeyService)

	mux.HandleFunc("GET /healthz", healthHandler)

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		readyHandler(w, r, db.Pool(), redisQueue)
	})

	mux.HandleFunc(
		"POST /api/v1/leads",
		apiKeyAuth(http.HandlerFunc(leadHandler.Create)).ServeHTTP,
	)

	mux.Handle("GET /api/v1/leads", apiKeyAuth(leadListHandler))
	mux.Handle("GET /admin/leads", apiKeyAuth(adminLeadsHandler))

	mux.HandleFunc("POST /api/v1/organizations", orgHandler.Create)

	mux.HandleFunc("POST /api/v1/webhooks", webhookHandler.Create)
	mux.HandleFunc("GET /api/v1/webhooks", webhookHandler.List)
	mux.HandleFunc("DELETE /api/v1/webhooks/", webhookHandler.Delete)

	mux.HandleFunc("POST /api/v1/api-keys", apiKeyHandler.Create)
	mux.HandleFunc("GET /api/v1/api-keys", apiKeyHandler.List)
	mux.HandleFunc("DELETE /api/v1/api-keys", apiKeyHandler.Delete)

	mux.HandleFunc(
		"GET /admin/webhooks/deliveries/stats",
		apiKeyAuth(http.HandlerFunc(webhookAdminHandler.Stats)).ServeHTTP,
	)

	mux.Handle("GET /metrics", promhttp.Handler())

	var httpHandler http.Handler = mux

	httpHandler = middleware.RequestLogging(logger)(httpHandler)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("api server started",
			slog.String("address", server.Addr),
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received",
			slog.String("signal", sig.String()),
		)

	case err := <-errCh:
		logger.Error("api server failed", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func readyHandler(
	w http.ResponseWriter,
	r *http.Request,
	db *pgxpool.Pool,
	redisQueue *queue.RedisQueue,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		http.Error(
			w,
			`{"status":"not_ready","dependency":"postgres"}`,
			http.StatusServiceUnavailable,
		)

		return
	}

	if err := redisQueue.Ping(ctx); err != nil {
		http.Error(
			w,
			`{"status":"not_ready","dependency":"redis"}`,
			http.StatusServiceUnavailable,
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ready"}`))
}
