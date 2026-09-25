package config

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"platform/backend/internal/filelink"
	"platform/backend/internal/models"
)

const (
	limenSecretLength = 32

	defaultParsingTimeout  = 90 * time.Second
	defaultBalanceCurrency = "BYN"

	defaultLowBalanceThreshold = 15.0
	defaultLowBalanceNotifyAt  = "12:48"
	defaultNotifyTimezone      = "Europe/Minsk"
)

type Config struct {
	Port           string
	Env            string
	ServiceName    string
	FrontendURL    string
	AllowedOrigins []string
	DatabaseURL    string
	BaseURL        string
	LimenSecret    string
	CookieSecure   bool

	LogLevel  string
	LogFormat string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	FilesBaseURL      string
	FileLinkSecret    string
	ParsingAPIURL     string
	ParsingAPIKey     string
	ParsingAPITimeout time.Duration
	BalanceCurrency   string

	TelegramBotToken string

	LowBalanceThresholdMinor int64
	LowBalanceNotifyAt       string
	NotifyLocation           *time.Location
}

func Load() (*Config, error) {
	if err := LoadEnv(); err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}
	port := GetEnv("PORT", "8080")
	cfg := &Config{
		Port:         port,
		Env:          GetEnv("APP_ENV", "local"),
		ServiceName:  GetEnv("SERVICE_NAME", "winforce-backend"),
		FrontendURL:  GetEnv("FRONTEND_URL", "http://localhost:5173"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		BaseURL:      GetEnv("APP_BASE_URL", "http://localhost:"+port),
		LimenSecret:  os.Getenv("LIMEN_SECRET"),
		CookieSecure: GetBoolEnv("COOKIE_SECURE", false),

		LogLevel:  GetEnv("LOG_LEVEL", "info"),
		LogFormat: GetEnv("LOG_FORMAT", "json"),

		MinioEndpoint:  GetEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: os.Getenv("MINIO_ROOT_USER"),
		MinioSecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioUseSSL:    GetBoolEnv("MINIO_USE_SSL", false),

		ParsingAPIURL:     strings.TrimRight(os.Getenv("PARSING_API_URL"), "/"),
		ParsingAPIKey:     os.Getenv("PARSING_API_KEY"),
		ParsingAPITimeout: GetDurationEnv("PARSING_API_TIMEOUT", defaultParsingTimeout),

		BalanceCurrency: strings.ToUpper(GetEnv("BALANCE_CURRENCY", defaultBalanceCurrency)),

		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),

		LowBalanceThresholdMinor: models.MinorFromFloat(GetFloatEnv("LOW_BALANCE_THRESHOLD", defaultLowBalanceThreshold)),
		LowBalanceNotifyAt:       GetEnv("LOW_BALANCE_NOTIFY_AT", defaultLowBalanceNotifyAt),
	}

	loc, err := time.LoadLocation(GetEnv("NOTIFY_TIMEZONE", defaultNotifyTimezone))
	if err != nil {
		return nil, fmt.Errorf("NOTIFY_TIMEZONE: %w", err)
	}
	cfg.NotifyLocation = loc

	cfg.AllowedOrigins = allowedOrigins(cfg.FrontendURL)
	cfg.FilesBaseURL = strings.TrimRight(
		GetEnv("FILES_BASE_URL", strings.TrimRight(cfg.BaseURL, "/")+filelink.DefaultPathPrefix),
		"/",
	)
	cfg.FileLinkSecret = GetEnv("FILE_LINK_SECRET", cfg.LimenSecret)

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	if len(cfg.LimenSecret) != limenSecretLength {
		return nil, fmt.Errorf("LIMEN_SECRET must be exactly %d bytes, got %d", limenSecretLength, len(cfg.LimenSecret))
	}

	return cfg, nil
}

func allowedOrigins(frontendURL string) []string {
	origins := []string{frontendURL}

	for origin := range strings.SplitSeq(os.Getenv("ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" || slices.Contains(origins, origin) {
			continue
		}
		origins = append(origins, origin)
	}
	return origins
}

func LoadEnv() error {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
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

func GetFloatEnv(key string, defaultValue float64) float64 {
	value, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func GetDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}
