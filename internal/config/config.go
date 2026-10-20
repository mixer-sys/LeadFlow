package config

import (
	"fmt"
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
}

func New() config {
	return config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "leadflow"),
		DBPassword: GetSecret("db_password", ""),
		DBName:     getEnv("DB_NAME", "leadflow"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),

		APIKey:    GetSecret("api_key", ""),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		GRPCInternalToken: GetSecret("grpc_internal_token", ""),
		OTLPEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultVal
}

func GetSecret(key, defaultVal string) string {
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

func GetSecretInt64(key string, defaultVal int64) int64 {
	value := GetSecret(key, "")

	if value == "" {
		return defaultVal
	}

	var result int64

	if _, err := fmt.Sscan(value, "%d, &result"); err != nil {
		return defaultVal
	}

	return result
}
