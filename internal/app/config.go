package app

import "os"

type Config struct {
	AppEnv             string
	HTTPAddr           string
	PublicBaseURL      string
	DatabaseURL        string
	RedisAddr          string
	ClickHouseAddr     string
	ClickHouseDatabase string
	DemoAPIKey         string
}

func LoadConfig() Config {
	return Config{
		AppEnv:             env("APP_ENV", "local"),
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		PublicBaseURL:      env("PUBLIC_BASE_URL", "http://localhost:8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://qr:qr@localhost:5432/qr?sslmode=disable"),
		RedisAddr:          env("REDIS_ADDR", "localhost:6379"),
		ClickHouseAddr:     env("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDatabase: env("CLICKHOUSE_DATABASE", "qr_analytics"),
		DemoAPIKey:         env("DEMO_API_KEY", "qk_demo_local_dev_key"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
