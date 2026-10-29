package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"leadflow/internal/config"
	"leadflow/internal/domain"
	log "leadflow/internal/logger"
	"leadflow/internal/platform/bitrix24"
	"leadflow/internal/platform/crypto"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/platform/telegram"
	"leadflow/internal/platform/telemetry"
	"leadflow/internal/repository"
	"leadflow/internal/service"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const maxTelegramRetries = 5

type leadProcessor struct {
	logger           *slog.Logger
	tracer           trace.Tracer
	leadRepo         *repository.LeadRepo
	webhookSvc       *service.WebhookService
	telegramChannels *service.TelegramChannelService
	bx               *bitrix24.Client
}

func main() {
	cfg := config.New()

	logger := log.New(log.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	slog.SetDefault(logger)

	ctx := context.Background()

	shutdownTracing, initTraceErr := telemetry.InitTracer(
		ctx,
		cfg.OTLPEndpoint,
	)
	if initTraceErr != nil {
		logger.Error("failed to initialize tracing", "error", initTraceErr)
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
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	defer func() {
		_ = db.Close(ctx)
	}()

	leadRepo := repository.NewLeadRepo(db.Pool())
	webhookRepo := repository.NewWebhookRepo(db.Pool())

	telegramChannelRepo := telegram.NewChannelRepository(db.Pool())

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		logger.Error("failed to create encryptor", "error", err)
		os.Exit(1)
	}

	telegramSender := telegram.NewSenderAdapter()

	telegramChannelSvc := service.NewTelegramChannelService(
		telegramChannelRepo,
		encryptor,
		telegramSender,
	)

	q := queue.NewRedisQueue(cfg.RedisAddr, "leads")

	if err := q.EnsureLeadGroup(ctx); err != nil {
		logger.Error(
			"failed to ensure redis consumer group",
			"error", err,
		)
		os.Exit(1)
	}

	var bx *bitrix24.Client

	if cfg.Bitrix24WebhookURL != "" {
		bx = bitrix24.NewClient(cfg.Bitrix24WebhookURL)
		logger.Info("bitrix24 client initialized")
	} else {
		logger.Warn(
			"bitrix24 not configured, leads will not be sent to CRM",
		)
	}

	tracer := otel.Tracer("leadflow-worker")

	webhookService := service.NewWebhookService(webhookRepo)

	processor := &leadProcessor{
		logger:           logger,
		tracer:           tracer,
		leadRepo:         leadRepo,
		webhookSvc:       webhookService,
		telegramChannels: telegramChannelSvc,
		bx:               bx,
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("worker started", "redis_addr", cfg.RedisAddr)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := retryDueWebhooks(ctx, processor); err != nil {
					logger.Error(
						"failed to retry webhooks",
						"error", err,
					)
				}
			}
		}
	}()

	for {
		select {
		case <-stopCh:
			logger.Info("worker shutdown signal received")
			return

		default:
			msgs, err := q.ReadLeadCreatedGroup(ctx, 10)
			if err != nil {
				logger.Error("failed to read events", "error", err)
				time.Sleep(time.Second)
				continue
			}

			if len(msgs) == 0 {
				continue
			}

			for _, m := range msgs {
				dataVal, ok := m.Values["payload"]
				if !ok {
					logger.Warn(
						"event payload field missing",
						"msg_id", m.ID,
						"fields", m.Values,
					)

					_ = q.AckLeadCreated(ctx, m.ID)
					continue
				}

				dataStr, ok := dataVal.(string)
				if !ok {
					logger.Warn(
						"event payload is not a string",
						"msg_id", m.ID,
						"payload_type", fmt.Sprintf("%T", dataVal),
					)

					_ = q.AckLeadCreated(ctx, m.ID)
					continue
				}

				var ev queue.LeadCreatedEvent

				if err := json.Unmarshal([]byte(dataStr), &ev); err != nil {
					logger.Error(
						"failed to unmarshal event",
						"error", err,
						"msg_id", m.ID,
					)

					_ = q.AckLeadCreated(ctx, m.ID)
					continue
				}

				if err := processor.processLeadCreated(ctx, m, ev); err != nil {
					logger.Error(
						"failed to process event",
						"error", err,
						"msg_id", m.ID,
						"lead_id", ev.LeadID,
					)

					select {
					case <-time.After(retryDelay()):
					case <-ctx.Done():
						return
					}

					continue
				}

				if err := q.AckLeadCreated(ctx, m.ID); err != nil {
					logger.Error(
						"failed to ack event",
						"error", err,
						"msg_id", m.ID,
					)
				}
			}
		}
	}
}

func retryDueWebhooks(
	ctx context.Context,
	p *leadProcessor,
) error {
	ctx, span := p.tracer.Start(ctx, "worker.retry-due-webhooks")
	defer span.End()

	deliveries, err := p.webhookSvc.GetDueDeliveries(ctx, 50)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get due deliveries")

		return err
	}

	for _, delivery := range deliveries {
		lead, err := p.leadRepo.GetByID(ctx, delivery.LeadID)
		if err != nil {
			p.logger.ErrorContext(
				ctx,
				"failed to retry webhook delivery",
				"error", err,
				"delivery_id", delivery.ID,
			)

			continue
		}

		webhook, err := p.webhookSvc.GetWebhookByID(
			ctx,
			delivery.WebhookID,
		)
		if err != nil {
			p.logger.ErrorContext(
				ctx,
				"failed to get webhook for retry",
				"error", err,
				"webhook_id", delivery.WebhookID,
			)

			continue
		}

		if err := p.webhookSvc.RetryDelivery(
			ctx,
			webhook,
			lead,
			&delivery,
		); err != nil {
			p.logger.ErrorContext(
				ctx,
				"failed to retry webhook delivery",
				"error", err,
				"delivery_id", delivery.ID,
			)
		}
	}

	return nil
}

func retryDelay() time.Duration {
	return 5 * time.Second
}

func splitName(fullName *string) (string, string) {
	if fullName == nil || *fullName == "" {
		return "", ""
	}

	parts := strings.SplitN(*fullName, " ", 2)

	if len(parts) == 1 {
		return parts[0], ""
	}

	return parts[0], parts[1]
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func (p *leadProcessor) processLeadCreated(
	ctx context.Context,
	m redis.XMessage,
	ev queue.LeadCreatedEvent,
) error {
	carrier := propagation.MapCarrier{
		"traceparent": ev.Traceparent,
	}

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	ctx, span := p.tracer.Start(
		ctx,
		"worker.process-lead-created",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	span.SetAttributes(
		attribute.Int64("lead.id", ev.LeadID),
		attribute.String("lead.request_id", ev.RequestID),
		attribute.String("redis.message_id", m.ID),
	)

	ctx = log.WithRequestID(ctx, ev.RequestID)

	p.logger.InfoContext(
		ctx,
		"received event",
		slog.String("request_id", ev.RequestID),
		slog.Int64("lead_id", ev.LeadID),
		slog.String("msg_id", m.ID),
	)

	lead, err := p.leadRepo.GetByID(ctx, ev.LeadID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get lead")

		return fmt.Errorf("get lead %d: %w", ev.LeadID, err)
	}

	if lead == nil {
		span.SetStatus(codes.Error, "lead not found")

		return fmt.Errorf("lead not found: lead_id %d", ev.LeadID)
	}

	if p.bx != nil {
		if lead.BitrixID == nil {
			ctx, bitrixSpan := p.tracer.Start(
				ctx,
				"bitrix24.create-lead",
				trace.WithSpanKind(trace.SpanKindClient),
			)

			name, lastName := splitName(lead.Name)

			bitrixID, err := p.bx.CreateLead(ctx, bitrix24.CreateLeadParams{
				Title: fmt.Sprintf(
					"Lead #%d from %s",
					lead.ID,
					lead.Source,
				),
				Name:     name,
				LastName: lastName,
				Phone:    deref(lead.Phone),
				Email:    deref(lead.Email),
				Comments: deref(lead.Message),
			})

			if err != nil {
				bitrixSpan.RecordError(err)
				bitrixSpan.SetStatus(
					codes.Error,
					"failed to create lead in bitrix24",
				)
				bitrixSpan.End()

				p.logger.ErrorContext(
					ctx,
					"bitrix24 sync failed, continuing to telegram",
					"error", err,
					"lead_id", lead.ID,
				)
			}

			bitrixSpan.SetAttributes(
				attribute.Int64("bitrix.lead_id", bitrixID),
			)
			bitrixSpan.SetStatus(codes.Ok, "lead created")
			bitrixSpan.End()

			if err := p.leadRepo.SetBitrixID(ctx, lead.ID, bitrixID); err != nil {
				return fmt.Errorf(
					"save bitrix id for lead %d: %w",
					lead.ID,
					err,
				)
			}
		}
	}

	ctx, webhookSpan := p.tracer.Start(ctx, "webhook.send")

	webhookErr := p.webhookSvc.SendWebhooks(ctx, lead, ev.RequestID)

	if webhookErr != nil {
		webhookSpan.RecordError(webhookErr)
		webhookSpan.SetStatus(codes.Error, "failed to send webhooks")

		p.logger.ErrorContext(
			ctx,
			"webhooks failed, continuing to telegram",
			"error", webhookErr,
			"lead_id", lead.ID,
		)
	} else {
		webhookSpan.SetStatus(codes.Ok, "webhooks sent")

		p.logger.InfoContext(
			ctx,
			"webhooks sent",
			"lead_id", lead.ID,
		)
	}

	webhookSpan.End()

	if err := p.notifyTelegram(ctx, lead, ev); err != nil {
		return err
	}

	return nil
}

func (p *leadProcessor) notifyTelegram(
	ctx context.Context,
	lead *domain.Lead,
	ev queue.LeadCreatedEvent,
) error {
	channels, err := p.telegramChannels.ListActive(
		ctx,
		lead.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf(
			"list active telegram channels for lead %d: %w",
			lead.ID,
			err,
		)
	}

	if len(channels) == 0 {
		p.logger.InfoContext(
			ctx,
			"no active telegram channels for organization, skipping notification",
			"lead_id", lead.ID,
			"organization_id", lead.OrganizationID,
		)

		return nil
	}

	if lead.TelegramMessageID != nil && *lead.TelegramMessageID != 0 {
		p.logger.InfoContext(
			ctx,
			"lead already notified, skipping",
			"lead_id", lead.ID,
		)

		return nil
	}

	if lead.TelegramRetryCount >= maxTelegramRetries {
		p.logger.WarnContext(
			ctx,
			"telegram retry limit reached",
			"lead_id", lead.ID,
		)

		return nil
	}

	ctx, telegramSpan := p.tracer.Start(
		ctx,
		"telegram.send-messages",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer telegramSpan.End()

	text := fmt.Sprintf(
		"🔔 Новая заявка\nLead ID: %d",
		ev.LeadID,
	)

	var sendErr error

	for _, channel := range channels {
		msgID, err := p.telegramChannels.SendMessage(
			ctx,
			channel.BotToken,
			channel.ChatID,
			text,
		)

		if err != nil {
			telegramSpan.RecordError(err)

			p.logger.ErrorContext(
				ctx,
				"failed to send telegram message",
				"error", err,
				"channel_id", channel.ID,
				"lead_id", lead.ID,
			)

			sendErr = err
			continue
		}

		telegramSpan.SetAttributes(
			attribute.Int64(
				fmt.Sprintf("telegram.channel_%d.message_id", channel.ID),
				msgID,
			),
		)

		p.logger.InfoContext(
			ctx,
			"telegram message sent",
			"lead_id", lead.ID,
			"channel_id", channel.ID,
			"message_id", msgID,
		)
	}

	if sendErr != nil {
		telegramSpan.SetStatus(
			codes.Error,
			"failed to send telegram messages",
		)

		if incrErr := p.leadRepo.IncrTelegramRetryCount(
			ctx,
			ev.LeadID,
			maxTelegramRetries,
			domain.LeadStatusFailed,
		); incrErr != nil {
			p.logger.ErrorContext(
				ctx,
				"failed to increment retry count",
				"error", incrErr,
				"lead_id", lead.ID,
			)
		}

		return fmt.Errorf(
			"send telegram messages for lead %d: %w",
			lead.ID,
			sendErr,
		)
	}

	telegramSpan.SetStatus(codes.Ok, "messages sent")

	return nil
}
