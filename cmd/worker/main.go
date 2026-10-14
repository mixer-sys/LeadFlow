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

	"leadflow/internal/domain"
	"leadflow/internal/middleware"
	"leadflow/internal/platform/bitrix24"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/platform/telegram"
	"leadflow/internal/repository"
	"leadflow/internal/service"

	"github.com/redis/go-redis/v9"
)

type config struct {
	RedisAddr        string
	TelegramBotToken string
	TelegramChatID   int64

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	Bitrix24WebhookURL string
}

func loadConfig() config {
	return config{
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnvInt64("TELEGRAM_CHAT_ID", 0),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "leadflow"),
		DBPassword: getEnv("DB_PASSWORD", "leadflow"),
		DBName:     getEnv("DB_NAME", "leadflow"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		Bitrix24WebhookURL: getEnv("BITRIX24_WEBHOOK_URL", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}

	var val int64
	if _, err := fmt.Sscanf(v, "%d", &val); err != nil {
		return defaultVal
	}

	return val
}

type LeadCreatedEvent struct {
	LeadID    int64  `json:"lead_id"`
	RequestID string `json:"request_id"`
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

	leadRepo := repository.NewLeadRepo(db.Pool())

	webhookRepo := repository.NewWebhookRepo(db.Pool())
	webhookService := service.NewWebhookService(webhookRepo)

	q := queue.NewRedisQueue(cfg.RedisAddr, "leads")

	var tg *telegram.Client
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != 0 {
		tg = telegram.NewClient(cfg.TelegramBotToken)
		logger.Info("telegram client initialized", "chat_id", cfg.TelegramChatID)
	} else {
		logger.Warn("telegram not configured, notifications will be skipped")
	}

	var bx *bitrix24.Client
	if cfg.Bitrix24WebhookURL != "" {
		bx = bitrix24.NewClient(cfg.Bitrix24WebhookURL)
		logger.Info("bitrix24 client initialized", "webhook_url", cfg.Bitrix24WebhookURL)
	} else {
		logger.Warn("bitrix24 not configured, leads will not be sent to CRM")
	}

	processor := &leadProcessor{
		logger:       logger,
		leadRepo:     leadRepo,
		webhookSvc:   webhookService,
		tg:           tg,
		bx:           bx,
		telegramChat: cfg.TelegramChatID,
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("worker started", "redis_addr", cfg.RedisAddr)

	lastID := "0"

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := retryDueWebhooks(ctx, processor); err != nil {
					logger.Error("failed to retry webhooks", "error", err)
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
			msgs, err := q.ReadLeadCreated(ctx, 10, lastID)
			if err != nil {
				logger.Error("failed to read events", "error", err)
				continue
			}

			if len(msgs) == 0 {
				continue
			}

			for _, m := range msgs {
				dataVal, ok := m.Values["payload"]
				if !ok {
					processor.logger.Warn("event payload field missing",
						"msg_id", m.ID,
						"fields", m.Values,
					)
					continue
				}

				dataStr, ok := dataVal.(string)
				if !ok {
					processor.logger.Warn(
						"event payload is not a string",
						"msg_id", m.ID,
						"payload_type", fmt.Sprintf("%T", dataVal),
					)
					continue
				}

				var ev queue.LeadCreatedEvent
				if err := json.Unmarshal([]byte(dataStr), &ev); err != nil {
					logger.Error("failed to unmarshal event", "error", err)
					continue
				}

				if err := processor.processLeadCreated(ctx, m, ev); err != nil {
					logger.Error("failed to process event",
						"error", err,
						"msg_id", m.ID,
					)

					delay := retryDelay()

					logger.Warn("retrying event after delay",
						"msg_id", m.ID,
						"delay", delay.String(),
					)

					select {
					case <-time.After(5 * time.Second):
					case <-ctx.Done():
						return
					}
					continue
				}

				lastID = m.ID
			}
		}
	}
}

func retryDueWebhooks(
	ctx context.Context,
	p *leadProcessor,
) error {
	deliveries, err := p.webhookSvc.GetDueDeliveries(ctx, 50)
	if err != nil {
		return err
	}

	var lead *domain.Lead
	for _, delivery := range deliveries {
		if lead, err = p.leadRepo.GetByID(ctx, delivery.LeadID); err != nil {
			p.logger.Error("failed to retry webhook delivery",
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
			p.logger.Error("failed to get webhook for retry",
				"error", err,
				"webhook_id", delivery.WebhookID)
			continue
		}

		err = p.webhookSvc.RetryDelivery(
			ctx,
			webhook,
			lead,
			&delivery,
		)
		if err != nil {
			p.logger.Error("failed to retry webhook delivery",
				"error", err,
				"delivery_id", delivery.ID)
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

const maxTelegramRetries = 5

type leadProcessor struct {
	logger       *slog.Logger
	leadRepo     *repository.LeadRepo
	webhookSvc   *service.WebhookService
	tg           *telegram.Client
	bx           *bitrix24.Client
	telegramChat int64
}

func (p *leadProcessor) processLeadCreated(
	ctx context.Context,
	m redis.XMessage,
	ev queue.LeadCreatedEvent,
) error {
	ctx = middleware.WithRequestID(ctx, ev.RequestID)
	p.logger.InfoContext(
		ctx,
		"received event",
		slog.String("request_id", ev.RequestID),
		slog.Int64("lead_id", ev.LeadID),
		slog.String("msg_id", m.ID),
	)

	lead, err := p.leadRepo.GetByID(ctx, ev.LeadID)
	if err != nil {
		p.logger.ErrorContext(
			ctx,
			"failed to get lead",
			slog.String("request_id", ev.RequestID),
			slog.Int64("lead_id", ev.LeadID),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("get lead %d: %w", ev.LeadID, err)
	}

	if lead == nil {
		p.logger.WarnContext(
			ctx,
			"lead not found",
			slog.String("request_id", ev.RequestID),
			slog.Int64("lead_id", ev.LeadID),
		)

		return fmt.Errorf("lead not found: lead_id %d", ev.LeadID)
	}

	// Bitrix24
	if p.bx != nil {
		if lead.BitrixID == nil {
			name, lastName := splitName(lead.Name)

			bitrixID, err := p.bx.CreateLead(ctx, bitrix24.CreateLeadParams{
				Title:    fmt.Sprintf("Lead #%d from %s", lead.ID, lead.Source),
				Name:     name,
				LastName: lastName,
				Phone:    deref(lead.Phone),
				Email:    deref(lead.Email),
				Comments: deref(lead.Message),
			})
			if err != nil {
				p.logger.Error("failed to create lead in bitrix24",
					"error", err,
					"lead_id", lead.ID,
				)
				return fmt.Errorf(
					"create bitrix lead for lead %d: %w",
					lead.ID,
					err,
				)
			}

			p.logger.Info("lead created in bitrix24",
				"lead_id", lead.ID,
				"bitrix_id", bitrixID,
			)

			if err := p.leadRepo.SetBitrixID(ctx, lead.ID, bitrixID); err != nil {
				return fmt.Errorf(
					"save bitrix id for lead %d: %w",
					lead.ID,
					err,
				)
			}
		} else {
			p.logger.Info("lead already exists in bitrix24, skipping",
				"lead_id", lead.ID,
				"bitrix_id", *lead.BitrixID,
			)
		}
	} else {
		p.logger.Warn("bitrix24 not configured, skipping CRM sync",
			"lead_id", lead.ID,
		)
	}

	// Webhooks
	if err := p.webhookSvc.SendWebhooks(ctx, lead, ev.RequestID); err != nil {
		p.logger.Error("failed to send webhooks",
			"error", err,
			"lead_id", lead.ID,
		)
		return fmt.Errorf("send webhooks for lead %d: %w", lead.ID, err)
	} else {
		p.logger.Info("webhooks sent", "lead_id", lead.ID)
	}

	// Telegram
	if p.tg == nil {
		p.logger.Warn("telegram not configured, skipping notification",
			"lead_id", lead.ID,
		)
		return nil
	}

	if lead.TelegramMessageID != nil && *lead.TelegramMessageID != 0 {
		p.logger.Info("lead already notified, skipping",
			"lead_id", lead.ID,
		)
		return nil
	}

	if lead.TelegramRetryCount >= maxTelegramRetries {
		p.logger.Error("telegram retry limit reached",
			"lead_id", lead.ID,
		)
		return nil
	}

	text := fmt.Sprintf("🔔 Новая заявка\nLead ID: %d", ev.LeadID)

	msgID, err := p.tg.SendMessage(ctx, telegram.SendMessageParams{
		ChatID: p.telegramChat,
		Text:   text,
	})
	if err != nil {
		p.logger.Error("failed to send telegram message",
			"error", err,
			"lead_id", lead.ID,
		)

		if incrErr := p.leadRepo.IncrTelegramRetryCount(
			ctx,
			ev.LeadID,
			maxTelegramRetries,
			domain.LeadStatusFailed,
		); incrErr != nil {
			p.logger.Error("failed to increment retry count",
				"error", incrErr,
				"lead_id", lead.ID,
			)
		}

		return fmt.Errorf(
			"send telegram message for lead %d: %w",
			lead.ID,
			err,
		)
	}

	p.logger.Info("telegram message sent",
		"lead_id", lead.ID,
		"message_id", msgID,
	)

	return nil
}
