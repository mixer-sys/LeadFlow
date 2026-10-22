package config

import (
	"os"
	"strings"
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

	APIKey    string
	LogLevel  string
	LogFormat string

	GRPCInternalToken string

	OTLPEndpoint string

	TelegramBotToken string
	TelegramChatID   string

	Bitrix24WebhookURL string
}

func New() config {
	return config{
		HTTPAddr: getValueOrSecret("HTTP_ADDR", "http_addr", ":8080"),

		DBHost:     getValueOrSecret("DB_HOST", "db_host", "localhost"),
		DBPort:     getValueOrSecret("DB_PORT", "db_port", "5432"),
		DBUser:     getValueOrSecret("DB_USER", "db_user", "leadflow"),
		DBPassword: getValueOrSecret("DB_PASSWORD", "db_password", ""),
		DBName:     getValueOrSecret("DB_NAME", "db_name", "leadflow"),
		DBSSLMode:  getValueOrSecret("DB_SSLMODE", "db_sslmode", "disable"),

		RedisAddr: getValueOrSecret("REDIS_ADDR", "redis_addr", "localhost:6379"),

		APIKey:    getValueOrSecret("API_KEY", "api_key", ""),
		LogLevel:  getValueOrSecret("LOG_LEVEL", "log_level", "info"),
		LogFormat: getValueOrSecret("LOG_FORMAT", "log_format", "json"),

		GRPCInternalToken: getValueOrSecret("GRPC_INTERNAL_TOKEN", "grpc_internal_token", ""),
		OTLPEndpoint:      getValueOrSecret("OTEL_EXPORTER_OTLP_ENDPOINT", "otel_exporter_otlp_endpoint", ""),

		TelegramBotToken: getValueOrSecret("TELETELEGRAM_BOT_TOKEN", "telegram_bot_token", ""),
		TelegramChatID:   getValueOrSecret("TELEGRAM_CHAT_ID", "telegram_chat_id", ""),

		Bitrix24WebhookURL: getValueOrSecret("BITRIX24_WEBHOOK_URL", "bitrix24_webhook_url", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultVal
}

func getSecret(key, defaultVal string) string {
	path := "/run/secrets/" + key

	data, err := os.ReadFile(path)
	if err == nil {
		value := strings.TrimSpace(string(data))

		if value != "" {
			return value
		}
	}

	return getEnv(key, defaultVal)
}

func getValueOrSecret(envKey, secretName, defaultVal string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if v := getSecret(secretName, ""); v != "" {
		return v
	}

	return defaultVal
}
