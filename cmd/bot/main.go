package main

import (
	"log"
	"os"

	"telegram_bot/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := &app.Config{
		Token:         mustGetEnv("BOT_TOKEN"),
		PostgresDSN:   mustGetEnv("POSTGRES_DSN"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPwd:      os.Getenv("REDIS_PASSWORD"),
		S3Endpoint:    mustGetEnv("S3_ENDPOINT"),
		S3AccessKey:   mustGetEnv("S3_ACCESS_KEY"),
		S3SecretKey:   mustGetEnv("S3_SECRET_KEY"),
		S3Bucket:      mustGetEnv("S3_BUCKET"),
		EncryptionKey: mustGetEnv("ENCRYPTION_KEY"),
		PathSalt:      mustGetEnv("PATH_SALT"),
	}

	app.Run(cfg)
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("environment variable %s is not set", key)
	}
	return val
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
