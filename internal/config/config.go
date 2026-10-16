package config

import "os"

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
		DBPassword: getEnv("DB_PASSWORD", "leadflow"),
		DBName:     getEnv("DB_NAME", "leadflow"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),

		APIKey:    getEnv("API_KEY", ""),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		GRPCInternalToken: getEnv("GRPC_INTERNAL_TOKEN", ""),
		OTLPEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultVal
}
