package config

import (
"testing"
"time"
)

func TestLoad_WithDefaultsAndRequiredEnv(t *testing.T) {
t.Setenv("TELEGRAM_BOT_TOKEN", "token")
t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
t.Setenv("JWT_SECRET", "test-secret")

cfg, err := Load()
if err != nil {
t.Fatalf("Load() returned error: %v", err)
}

if cfg.Port != "8080" {
t.Errorf("Port = %q, want 8080", cfg.Port)
}
if cfg.Environment != "development" {
t.Errorf("Environment = %q, want development", cfg.Environment)
}
if cfg.MongoDatabase != "habbit" {
t.Errorf("MongoDatabase = %q, want habbit", cfg.MongoDatabase)
}
if cfg.JWTExpiration != 7*24*time.Hour {
t.Errorf("JWTExpiration = %v, want %v", cfg.JWTExpiration, 7*24*time.Hour)
}
if cfg.RateLimitMax != 100 {
t.Errorf("RateLimitMax = %d, want 100", cfg.RateLimitMax)
}
if cfg.RateLimitWindow != time.Minute {
t.Errorf("RateLimitWindow = %v, want %v", cfg.RateLimitWindow, time.Minute)
}
if cfg.CacheDefaultTTL != 5*time.Minute {
t.Errorf("CacheDefaultTTL = %v, want %v", cfg.CacheDefaultTTL, 5*time.Minute)
}
}

func TestLoad_WithEnvOverrides(t *testing.T) {
t.Setenv("TELEGRAM_BOT_TOKEN", "token")
t.Setenv("MONGODB_URI", "mongodb://example:27017")
t.Setenv("MONGODB_DATABASE", "custom_db")
t.Setenv("PORT", "9090")
t.Setenv("ENVIRONMENT", "staging")
t.Setenv("JWT_SECRET", "custom-secret")
t.Setenv("JWT_EXPIRATION", "24h")
t.Setenv("RATE_LIMIT_MAX", "42")
t.Setenv("RATE_LIMIT_WINDOW", "30s")
t.Setenv("CACHE_DEFAULT_TTL", "10m")
t.Setenv("ALLOWED_ORIGINS", "https://example.com,https://app.example.com")

cfg, err := Load()
if err != nil {
t.Fatalf("Load() returned error: %v", err)
}

if cfg.Port != "9090" {
t.Errorf("Port = %q, want 9090", cfg.Port)
}
if cfg.Environment != "staging" {
t.Errorf("Environment = %q, want staging", cfg.Environment)
}
if cfg.MongoURI != "mongodb://example:27017" {
t.Errorf("MongoURI = %q", cfg.MongoURI)
}
if cfg.MongoDatabase != "custom_db" {
t.Errorf("MongoDatabase = %q", cfg.MongoDatabase)
}
if cfg.JWTExpiration != 24*time.Hour {
t.Errorf("JWTExpiration = %v, want 24h", cfg.JWTExpiration)
}
if cfg.RateLimitMax != 42 {
t.Errorf("RateLimitMax = %d, want 42", cfg.RateLimitMax)
}
if cfg.RateLimitWindow != 30*time.Second {
t.Errorf("RateLimitWindow = %v, want 30s", cfg.RateLimitWindow)
}
if cfg.CacheDefaultTTL != 10*time.Minute {
t.Errorf("CacheDefaultTTL = %v, want 10m", cfg.CacheDefaultTTL)
}
if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "https://example.com,https://app.example.com" {
t.Errorf("AllowedOrigins = %#v, expected single comma-separated value", cfg.AllowedOrigins)
}
}

func TestValidate(t *testing.T) {
tests := []struct {
name    string
cfg     Config
wantErr bool
}{
{
name: "valid config",
cfg: Config{
TelegramBotToken: "token",
MongoURI:         "mongodb://localhost:27017",
JWTSecret:        "secure-secret",
Environment:      "production",
},
wantErr: false,
},
{
name: "missing telegram token",
cfg: Config{
MongoURI:    "mongodb://localhost:27017",
JWTSecret:   "secure-secret",
Environment: "development",
},
wantErr: true,
},
{
name: "missing mongo uri",
cfg: Config{
TelegramBotToken: "token",
MongoURI:         "",
JWTSecret:        "secure-secret",
Environment:      "development",
},
wantErr: true,
},
{
name: "default jwt secret in production",
cfg: Config{
TelegramBotToken: "token",
MongoURI:         "mongodb://localhost:27017",
JWTSecret:        "your-secret-key-change-this",
Environment:      "production",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.cfg.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestEnvHelpers(t *testing.T) {
t.Setenv("TEST_ENV", "value")
if got := getEnv("TEST_ENV", "default"); got != "value" {
t.Errorf("getEnv() = %q, want value", got)
}
if got := getEnv("MISSING_TEST_ENV", "default"); got != "default" {
t.Errorf("getEnv() = %q, want default", got)
}

t.Setenv("TEST_INT", "123")
if got := getIntEnv("TEST_INT", 1); got != 123 {
t.Errorf("getIntEnv() = %d, want 123", got)
}
t.Setenv("TEST_INT", "bad")
if got := getIntEnv("TEST_INT", 1); got != 1 {
t.Errorf("getIntEnv() should fall back to default, got %d", got)
}

t.Setenv("TEST_DUR", "15s")
if got := getDurationEnv("TEST_DUR", time.Minute); got != 15*time.Second {
t.Errorf("getDurationEnv() = %v, want 15s", got)
}
t.Setenv("TEST_DUR", "bad")
if got := getDurationEnv("TEST_DUR", time.Minute); got != time.Minute {
t.Errorf("getDurationEnv() should fall back to default, got %v", got)
}

t.Setenv("TEST_SLICE", "a,b")
got := getSliceEnv("TEST_SLICE", []string{"x"})
if len(got) != 1 || got[0] != "a,b" {
t.Errorf("getSliceEnv() = %#v, want []string{\"a,b\"}", got)
}

got = getSliceEnv("MISSING_TEST_SLICE", []string{"x"})
if len(got) != 1 || got[0] != "x" {
t.Errorf("getSliceEnv() default = %#v, want []string{\"x\"}", got)
}
}
