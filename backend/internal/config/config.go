package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)


const limenSecretLength = 32

type Config struct {
	Port         string
	FrontendURL  string
	DatabaseURL  string
	BaseURL      string
	LimenSecret  string
	CookieSecure bool
}

func Load() (*Config, error) {
	if err:=LoadEnv(); err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}
	port := GetEnv("PORT", "8080")
	cfg := &Config{
		Port:         port,
		FrontendURL:  GetEnv("FRONTEND_URL", "http://localhost:5173"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		BaseURL:      GetEnv("APP_BASE_URL", "http://localhost:"+port),
		LimenSecret:  os.Getenv("LIMEN_SECRET"),
		CookieSecure: GetBoolEnv("COOKIE_SECURE", false),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	if len(cfg.LimenSecret) != limenSecretLength {
		return nil, fmt.Errorf("LIMEN_SECRET must be exactly %d bytes, got %d", limenSecretLength, len(cfg.LimenSecret))
	}

	return cfg, nil
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}
	return nil
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetBoolEnv(key string, defaultValue bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return defaultValue
	}
	return value
}
