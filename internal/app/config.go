package app

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv                      string
	HTTPAddr                    string
	PublicBaseURL               string
	DatabaseURL                 string
	RedisAddr                   string
	ClickHouseAddr              string
	ClickHouseDatabase          string
	ClickHousePassword          string
	DemoAPIKey                  string
	OwnerRateLimit              int
	RedirectCacheEnabled        bool
	AnalyticsWorkerEnabled      bool
	AnalyticsBatchSize          int
	AnalyticsBlockSeconds       int
	AnalyticsReclaimIdleSeconds int
}

func LoadConfig() Config {
	return Config{
		AppEnv:                      env("APP_ENV", "local"),
		HTTPAddr:                    env("HTTP_ADDR", ":8080"),
		PublicBaseURL:               env("PUBLIC_BASE_URL", "http://localhost:8080"),
		DatabaseURL:                 env("DATABASE_URL", "postgres://qr:qr@localhost:5432/qr?sslmode=disable"),
		RedisAddr:                   env("REDIS_ADDR", "localhost:6379"),
		ClickHouseAddr:              env("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDatabase:          env("CLICKHOUSE_DATABASE", "qr_analytics"),
		ClickHousePassword:          env("CLICKHOUSE_PASSWORD", "qr"),
		DemoAPIKey:                  env("DEMO_API_KEY", "qk_demo_local_dev_key"),
		OwnerRateLimit:              intEnv("OWNER_RATE_LIMIT_PER_MINUTE", 60),
		RedirectCacheEnabled:        boolEnv("REDIRECT_CACHE_ENABLED", true),
		AnalyticsWorkerEnabled:      boolEnv("ANALYTICS_WORKER_ENABLED", true),
		AnalyticsBatchSize:          intEnv("ANALYTICS_BATCH_SIZE", 500),
		AnalyticsBlockSeconds:       intEnv("ANALYTICS_BLOCK_SECONDS", 2),
		AnalyticsReclaimIdleSeconds: intEnv("ANALYTICS_RECLAIM_IDLE_SECONDS", 30),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
