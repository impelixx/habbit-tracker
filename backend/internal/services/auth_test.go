package services

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGenerateAndValidateJWT(t *testing.T) {
	authService := NewAuthService("test_secret_key", 7*24*time.Hour, "test_bot_token")

	userID := primitive.NewObjectID()
	telegramID := int64(123456789)

	// Generate token
	token, err := authService.GenerateJWT(userID, telegramID)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	// Validate token
	claims, err := authService.ValidateJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if claims.UserID != userID.Hex() {
		t.Errorf("Expected UserID %s, got %s", userID.Hex(), claims.UserID)
	}

	if claims.TelegramID != telegramID {
		t.Errorf("Expected TelegramID %d, got %d", telegramID, claims.TelegramID)
	}
}

func TestValidateJWTInvalid(t *testing.T) {
	authService := NewAuthService("test_secret_key", 7*24*time.Hour, "test_bot_token")

	// Test with invalid token
	_, err := authService.ValidateJWT("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// Test with empty token
	_, err = authService.ValidateJWT("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestValidateJWTExpired(t *testing.T) {
	// Create service with very short expiration
	authService := NewAuthService("test_secret_key", 1*time.Millisecond, "test_bot_token")

	userID := primitive.NewObjectID()
	telegramID := int64(123456789)

	token, err := authService.GenerateJWT(userID, telegramID)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = authService.ValidateJWT(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestValidateJWTWrongSecret(t *testing.T) {
	authService1 := NewAuthService("secret1", 7*24*time.Hour, "test_bot_token")
	authService2 := NewAuthService("secret2", 7*24*time.Hour, "test_bot_token")

	userID := primitive.NewObjectID()
	telegramID := int64(123456789)

	// Generate token with service1
	token, err := authService1.GenerateJWT(userID, telegramID)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Try to validate with service2 (different secret)
	_, err = authService2.ValidateJWT(token)
	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestValidateTelegramInitData(t *testing.T) {
	// Note: This test would require a real Telegram initData
	// For now, we'll just test that the function exists and handles invalid data
	authService := NewAuthService("test_secret_key", 7*24*time.Hour, "test_bot_token")

	// Test with empty initData
	_, err := authService.ValidateTelegramInitData("")
	if err == nil {
		t.Error("Expected error for empty initData")
	}

	// Test with invalid format
	_, err = authService.ValidateTelegramInitData("invalid_data")
	if err == nil {
		t.Error("Expected error for invalid initData format")
	}
}
