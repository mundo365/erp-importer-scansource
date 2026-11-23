package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	RabbitMQURL string
	HealthAddr  string
}

func (c Config) S3Configured() bool {
	return c.S3Bucket != "" && c.S3AccessKey != "" && c.S3SecretKey != ""
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rabbitURL == "" {
		return Config{}, fmt.Errorf("RABBITMQ_URL is required")
	}
	healthAddr := strings.TrimSpace(os.Getenv("HEALTH_ADDR"))
	if healthAddr == "" {
		healthAddr = ":8081"
	}
	cfg := Config{
		DatabaseURL: databaseURL,
		S3Endpoint:  strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		S3Region:    strings.TrimSpace(os.Getenv("S3_REGION")),
		S3Bucket:    strings.TrimSpace(os.Getenv("S3_BUCKET")),
		S3AccessKey: strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		S3SecretKey: strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		RabbitMQURL: rabbitURL,
		HealthAddr:  healthAddr,
	}
	if !cfg.S3Configured() {
		return Config{}, fmt.Errorf("S3_BUCKET, S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	return cfg, nil
}
