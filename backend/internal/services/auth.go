package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret     string
	jwtExpiration time.Duration
	botToken      string
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, jwtExpiration time.Duration, botToken string) *AuthService {
	return &AuthService{
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
		botToken:      botToken,
	}
}

// Claims represents JWT claims
type Claims struct {
	UserID     string `json:"userId"`
	TelegramID int64  `json:"telegramId"`
	jwt.RegisteredClaims
}

// ValidateTelegramInitData validates Telegram WebApp initData
func (s *AuthService) ValidateTelegramInitData(initData string) (map[string]string, error) {
	// Parse initData
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initData: %w", err)
	}

	// Extract hash
	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return nil, errors.New("hash not found in initData")
	}

	// Remove hash from values
	values.Del("hash")

	// Create data check string
	var dataCheckArr []string
	for key := range values {
		value := values.Get(key)
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(dataCheckArr)
	dataCheckString := strings.Join(dataCheckArr, "\n")

	// Create secret key
	secretKey := sha256.Sum256([]byte(s.botToken))

	// Calculate HMAC
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// Compare hashes
	if calculatedHash != receivedHash {
		return nil, errors.New("invalid hash: data may have been tampered with")
	}

	// Convert values to map
	result := make(map[string]string)
	for key := range values {
		result[key] = values.Get(key)
	}

	return result, nil
}

// GenerateJWT generates a JWT token for a user
func (s *AuthService) GenerateJWT(userID primitive.ObjectID, telegramID int64) (string, error) {
	claims := Claims{
		UserID:     userID.Hex(),
		TelegramID: telegramID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the claims
func (s *AuthService) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
