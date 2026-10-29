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
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	leadflowv1 "leadflow/api/gen/leadflow/v1"
	"leadflow/internal/auth"
	config "leadflow/internal/config"
	"leadflow/internal/handler"
	grpchandler "leadflow/internal/handler/grpc"
	"leadflow/internal/logger"
	"leadflow/internal/middleware"
	grpcmiddleware "leadflow/internal/middleware/grpc"
	"leadflow/internal/platform/crypto"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/platform/telegram"
	"leadflow/internal/platform/telemetry"
	"leadflow/internal/repository"
	"leadflow/internal/service"
)

func main() {
	cfg := config.New()

	log := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	slog.SetDefault(log)

	log.Info("starting leadflow",
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("log_level", cfg.LogLevel),
		slog.String("log_format", cfg.LogFormat),
	)

	ctx := context.Background()

	shutdownTracing, err := telemetry.InitTracer(ctx, cfg.OTLPEndpoint)
	if err != nil {
		log.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer shutdownTracing(context.Background())

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
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	redisQueue := queue.NewRedisQueue(cfg.RedisAddr, "leads")

	grpcServer := newGRPCServer(&cfg, db, log)

	grpcListener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Error("failed to listen grpc", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info(
			"grpc server started",
			slog.String("address", grpcListener.Addr().String()),
		)

		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("grpc server failed", "error", err)
		}
	}()

	httpServer := newHTTPServer(&cfg, db, redisQueue, log)

	errCh := make(chan error, 1)

	go func() {
		log.Info(
			"api server started",
			slog.String("address", httpServer.Addr),
		)

		if err := httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		log.Info(
			"shutdown signal received",
			slog.String("signal", sig.String()),
		)

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		_ = httpServer.Shutdown(shutdownCtx)
		grpcServer.GracefulStop()

	case err := <-errCh:
		log.Error("api server failed", "error", err)
		os.Exit(1)
	}

	log.Info("api server stopped")
}

func newGRPCServer(
	cfg *config.Config,
	db *database.Postgres,
	log *slog.Logger,
) *grpc.Server {
	leadRepo := repository.NewLeadRepo(db.Pool())
	leadGRPC := grpchandler.NewLeadHandler(leadRepo, log)

	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcmiddleware.RequestIDUnaryServerInterceptor(log),
			grpcmiddleware.AuthUnaryServerInterceptor(
				log,
				cfg.GRPCInternalToken,
			),
		),
	)

	leadflowv1.RegisterLeadServiceServer(server, leadGRPC)
	reflection.Register(server)

	return server
}

func newHTTPServer(
	cfg *config.Config,
	db *database.Postgres,
	redisQueue *queue.RedisQueue,
	log *slog.Logger,
) *http.Server {
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Error("failed to create encryptor", "error", err)
		os.Exit(1)
	}

	telegramSender := telegram.NewSenderAdapter()

	leadRepo := repository.NewLeadRepo(db.Pool())
	orgRepo := repository.NewOrganizationRepo(db.Pool())
	webhookRepo := repository.NewWebhookRepo(db.Pool())
	apiKeyRepo := repository.NewAPIKeyRepo(db.Pool())
	userRepo := repository.NewUserRepo(db.Pool())

	telegramChannelRepo := telegram.NewChannelRepository(db.Pool())

	leadSvc := service.NewLeadService(leadRepo, redisQueue)
	orgSvc := service.NewOrganizationService(orgRepo)
	webhookSvc := service.NewWebhookService(webhookRepo)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	authService := service.NewAuthService(userRepo)

	telegramChannelSvc := service.NewTelegramChannelService(
		telegramChannelRepo,
		encryptor,
		telegramSender,
	)

	leadHandler := handler.NewLeadHandler(leadSvc)
	leadListHandler := handler.NewLeadListHandler(leadSvc)
	orgHandler := handler.NewOrganizationHandler(orgSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
	webhookAdminHandler := handler.NewWebhookAdminHandler(webhookSvc)
	authHandler := handler.NewAuthHandler(authService)

	webhookTestSvc := service.NewWebhookTestService()

	webhookTestHandler := handler.NewWebhookTestHandler(webhookTestSvc)

	pagesHandler, err := handler.NewPagesHandler()
	if err != nil {
		log.Error("failed to create pages handler", "error", err)
		os.Exit(1)
	}

	adminLeadsHandler, err := handler.NewAdminLeadsHandler(leadSvc)
	if err != nil {
		log.Error("failed to create admin leads handler", "error", err)
		os.Exit(1)
	}

	telegramChannelHandler, err := handler.NewTelegramChannelsHandler(
		telegramChannelSvc,
	)
	if err != nil {
		log.Error(
			"failed to create telegram channels handler",
			"error", err,
		)
		os.Exit(1)
	}

	healthHandler := handler.NewHealthHandler(db.Pool(), redisQueue)

	requireSession := handler.RequireSession(authService)

	webhookAdminRepo := repository.NewWebhookAdminRepo(db.Pool())
	webhookAdminSvc := service.NewWebhookAdminService(webhookAdminRepo)

	webhookSettingsHandler, err := handler.NewWebhookSettingsHandler(
		webhookAdminSvc,
	)
	if err != nil {
		log.Error("failed to create webhook settings handler", "error", err)
		os.Exit(1)
	}

	router := handler.NewRouter(handler.RouterDeps{
		Lead:         leadHandler,
		LeadList:     leadListHandler,
		Organization: orgHandler,
		Webhook:      webhookHandler,
		APIKey:       apiKeyHandler,
		WebhookAdmin: webhookAdminHandler,
		AdminLeads:   adminLeadsHandler,
		Health:       healthHandler,
		Auth:         authHandler,

		APIKeyAuth:     auth.APIKeyAuth(apiKeySvc),
		RequireSession: requireSession,

		Pages:            pagesHandler,
		TelegramChannels: telegramChannelHandler,
		WebhookSettings:  webhookSettingsHandler,
		WebhookTest:      webhookTestHandler,
	})

	var httpHandler http.Handler = router

	httpHandler = otelhttp.NewHandler(httpHandler, "leadflow-http")
	httpHandler = middleware.RequestLogging(log)(httpHandler)

	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
