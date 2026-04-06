package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// MongoDB
	MongoURI      string
	MongoDatabase string

	// Telegram
	TelegramBotToken   string
	TelegramWebhookURL string

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// OpenRouter
	OpenRouterAPIKey        string
	OpenRouterLLMModel      string
	OpenRouterLLMFallback   string

	// CORS
	AllowedOrigins []string

	// Rate Limiting
	RateLimitMax    int
	RateLimitWindow time.Duration

	// Cache
	CacheDefaultTTL time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		MongoURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGODB_DATABASE", "habbit"),

		TelegramBotToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhookURL: getEnv("TELEGRAM_WEBHOOK_URL", ""),

		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-this"),
		JWTExpiration: getDurationEnv("JWT_EXPIRATION", 7*24*time.Hour), // 7 days

		OpenRouterAPIKey:      getEnv("OPENROUTER_API_KEY", ""),
		OpenRouterLLMModel:    getEnv("OPENROUTER_LLM_MODEL", "deepseek/deepseek-chat"),
		OpenRouterLLMFallback: getEnv("OPENROUTER_LLM_FALLBACK", "mistralai/mistral-7b-instruct"),

		AllowedOrigins:  getSliceEnv("ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		RateLimitMax:    getIntEnv("RATE_LIMIT_MAX", 100),
		RateLimitWindow: getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),
		CacheDefaultTTL: getDurationEnv("CACHE_DEFAULT_TTL", 5*time.Minute),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if required configuration is set
func (c *Config) Validate() error {
	if c.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if c.MongoURI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}
	if c.JWTSecret == "your-secret-key-change-this" && c.Environment == "production" {
		return fmt.Errorf("JWT_SECRET must be changed in production")
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
