package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	APIPort string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	RabbitHost     string
	RabbitPort     string
	RabbitUser     string
	RabbitPassword string
	RabbitQueue    string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioRegion    string
	MinioUseSSL    bool

	TaskLeaseSeconds int
	TaskMaxRetries   int
}

func Load() (*Config, error) {
	cfg := &Config{}

	cfg.APIPort = getEnv("API_PORT", "8080")

	cfg.PostgresHost = getEnv("POSTGRES_HOST", "postgres")
	cfg.PostgresPort = getEnv("POSTGRES_PORT", "5432")
	cfg.PostgresUser = getEnv("POSTGRES_USER", "app")
	cfg.PostgresPassword = getEnv("POSTGRES_PASSWORD", "app")
	cfg.PostgresDB = getEnv("POSTGRES_DB", "app")

	cfg.RabbitHost = getEnv("RABBITMQ_HOST", "rabbitmq")
	cfg.RabbitPort = getEnv("RABBITMQ_PORT", "5672")
	cfg.RabbitUser = getEnv("RABBITMQ_USER", "guest")
	cfg.RabbitPassword = getEnv("RABBITMQ_PASSWORD", "guest")
	cfg.RabbitQueue = getEnv("RABBITMQ_QUEUE", "processing_tasks")

	cfg.MinioEndpoint = getEnv("MINIO_ENDPOINT", "http://minio:9000")
	cfg.MinioAccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinioSecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	cfg.MinioBucket = getEnv("MINIO_BUCKET", "media")
	cfg.MinioRegion = getEnv("MINIO_REGION", "us-east-1")
	cfg.MinioUseSSL = getEnvBool("MINIO_USE_SSL", false)

	cfg.TaskLeaseSeconds = getEnvInt("TASK_LEASE_SECONDS", 60)
	cfg.TaskMaxRetries = getEnvInt("TASK_MAX_RETRIES", 4)

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
	)
}

func (c *Config) RabbitURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		c.RabbitUser,
		c.RabbitPassword,
		c.RabbitHost,
		c.RabbitPort,
	)
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return def
}
