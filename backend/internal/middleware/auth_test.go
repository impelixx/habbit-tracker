package middleware

import (
"encoding/json"
"net/http/httptest"
"testing"
"time"

"github.com/gofiber/fiber/v2"
"github.com/golang-jwt/jwt/v5"
"github.com/impelixx/habbit-tracker/backend/internal/services"
"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAuthMiddleware(t *testing.T) {
authService := services.NewAuthService("test-secret", time.Hour, "bot-token")
userID := primitive.NewObjectID()
validToken, err := authService.GenerateJWT(userID, 12345)
if err != nil {
t.Fatalf("GenerateJWT failed: %v", err)
}

invalidUserClaims := services.Claims{
UserID:     "invalid-object-id",
TelegramID: 12345,
RegisteredClaims: jwt.RegisteredClaims{
ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
IssuedAt:  jwt.NewNumericDate(time.Now()),
},
}
invalidUserToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, invalidUserClaims).SignedString([]byte("test-secret"))
if err != nil {
t.Fatalf("failed to sign invalid user token: %v", err)
}

tests := []struct {
name       string
authHeader string
wantStatus int
wantError  string
}{
{name: "missing auth header", authHeader: "", wantStatus: fiber.StatusUnauthorized, wantError: "Missing authorization header"},
{name: "invalid auth header format", authHeader: "Token abc", wantStatus: fiber.StatusUnauthorized, wantError: "Invalid authorization header format"},
{name: "invalid token", authHeader: "Bearer invalid.token", wantStatus: fiber.StatusUnauthorized, wantError: "Invalid or expired token"},
{name: "invalid user id in token", authHeader: "Bearer " + invalidUserToken, wantStatus: fiber.StatusUnauthorized, wantError: "Invalid user ID in token"},
{name: "valid token", authHeader: "Bearer " + validToken, wantStatus: fiber.StatusOK, wantError: ""},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
app := fiber.New()
app.Get("/protected", AuthMiddleware(authService), func(c *fiber.Ctx) error {
uid := c.Locals("userId").(primitive.ObjectID)
tid := c.Locals("telegramId").(int64)
return c.JSON(fiber.Map{"userId": uid.Hex(), "telegramId": tid})
})

req := httptest.NewRequest("GET", "/protected", nil)
if tt.authHeader != "" {
req.Header.Set("Authorization", tt.authHeader)
}

resp, err := app.Test(req, -1)
if err != nil {
t.Fatalf("app.Test failed: %v", err)
}

if resp.StatusCode != tt.wantStatus {
t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
}

var body map[string]interface{}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("failed to decode body: %v", err)
}

if tt.wantStatus == fiber.StatusOK {
if body["userId"] != userID.Hex() {
t.Errorf("userId = %v, want %s", body["userId"], userID.Hex())
}
if body["telegramId"] != float64(12345) { // json number decoding
t.Errorf("telegramId = %v, want 12345", body["telegramId"])
}
return
}

if body["error"] != tt.wantError {
t.Errorf("error = %v, want %q", body["error"], tt.wantError)
}
})
}
}
