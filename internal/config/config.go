package config

import (
	"os"
)

// Config holds all environment-driven settings for the service.
type Config struct {
	AppPort       string
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	JWTExpiryHrs  int
	S3Bucket      string
	S3Region      string
	AWSAccessKey  string
	AWSSecretKey  string
	PythonSvcURL  string // internal URL of the Python RAG/transcription microservice
	Env           string
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// Load reads configuration from environment variables (populated via .env in dev).
func Load() *Config {
	return &Config{
		AppPort:      getEnv("APP_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ai_video?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		S3Bucket:     getEnv("S3_BUCKET", "ai-video-assistant-storage"),
		S3Region:     getEnv("S3_REGION", "ap-south-1"),
		AWSAccessKey: getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		PythonSvcURL: getEnv("PYTHON_SERVICE_URL", "http://python-service:8000"),
		Env:          getEnv("APP_ENV", "development"),
		JWTExpiryHrs: 24,
	}
}
