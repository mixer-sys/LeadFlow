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

	"leadflow/internal/platform/bitrix24"
	"leadflow/internal/platform/database"
	"leadflow/internal/platform/queue"
	"leadflow/internal/platform/telegram"
	"leadflow/internal/repository"
	"leadflow/internal/service"
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
	LeadID int64 `json:"lead_id"`
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

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("worker started", "redis_addr", cfg.RedisAddr)

	lastID := "0"

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
				dataVal, ok := m.Values["data"]
				if !ok {
					continue
				}

				dataStr, ok := dataVal.(string)
				if !ok {
					continue
				}

				dataStr, ok = dataVal.(string)
				if !ok {
					continue
				}

				var ev queue.LeadCreatedEvent
				if err := json.Unmarshal([]byte(dataStr), &ev); err != nil {
					logger.Error("failed to unmarshal event", "error", err)
					continue
				}

				logger.Info("received event", "lead_id", ev.LeadID, "msg_id", m.ID)

				lead, err := leadRepo.GetByID(ctx, ev.LeadID)
				if err != nil {
					logger.Error("failed to get lead", "error", err, "lead_id", ev.LeadID)
					continue
				}

				if lead == nil {
					logger.Warn("lead not found", "lead_id", ev.LeadID)
					continue
				}

				if lead.TelegramMessageID != nil && *lead.TelegramMessageID != 0 {
					logger.Info("lead already notified, skipping", "lead_id", ev.LeadID, "telegram_message_id", *lead.TelegramMessageID)
					lastID = m.ID
					continue
				}

				if err != nil {

				}

				if tg != nil {
					text := fmt.Sprintf("🔔 Новая заявка\nLead ID: %d", ev.LeadID)

					msgID, err := tg.SendMessage(ctx, telegram.SendMessageParams{
						ChatID: cfg.TelegramChatID,
						Text:   text,
					})
					if err != nil {
						logger.Error("failed to send telegram message", "error", err, "lead_id", ev.LeadID)

						logger.Error("failed to send telegram message", "error", err, "lead_id", ev.LeadID)

						const maxRetries = 5
						if err := leadRepo.IncrTelegramRetryCount(ctx, ev.LeadID, maxRetries); err != nil {
							logger.Error("failed to increment retry count", "error", err, "lead_id", ev.LeadID)
						}

						continue
					}

					logger.Info("telegram message sent", "lead_id", ev.LeadID, "message_id", msgID)

					if bx != nil {
						name, lastName := splitName(lead.Name)

						leadID, err := bx.CreateLead(ctx, bitrix24.CreateLeadParams{
							Title:    fmt.Sprintf("Лид #%d из %s", lead.ID, lead.Source),
							Name:     name,
							LastName: lastName,
							Phone:    deref(lead.Phone),
							Email:    deref(lead.Email),
							Comments: deref(lead.Message),
						})
						if err != nil {
							logger.Error("failed to create lead in bitrix24", "error", err, "lead_id", lead.ID)
						} else {
							logger.Info("lead created in bitrix24", "lead_id", lead.ID, "bitrix_id", leadID)
						}
					}

					if err := webhookService.SendWebhooks(ctx, lead); err != nil {
						logger.Error("failed to send webhooks", "error", err, "lead_id", lead.ID)
					} else {
						logger.Info("webhooks sent", "lead_id", lead.ID)
					}

					now := time.Now()

					if err := leadRepo.UpdateTelegramSent(ctx, ev.LeadID, msgID, now); err != nil {
						logger.Error("failed to update lead in database", "error", err, "lead_id", ev.LeadID)
					} else {
						logger.Info("lead updated in database", "lead_id", ev.LeadID, "message_id", msgID)
					}
				}

				lastID = m.ID

				// sent TG Message
			}
		}
	}
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
