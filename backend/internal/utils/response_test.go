package utils

import (
"encoding/json"
"net/http/httptest"
"testing"

"github.com/gofiber/fiber/v2"
)

func TestSuccessResponse(t *testing.T) {
app := fiber.New()
app.Get("/", func(c *fiber.Ctx) error {
return SuccessResponse(c, fiber.Map{"foo": "bar"})
})

req := httptest.NewRequest("GET", "/", nil)
resp, err := app.Test(req, -1)
if err != nil {
t.Fatalf("app.Test failed: %v", err)
}

var body map[string]interface{}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode response failed: %v", err)
}

if ok, _ := body["success"].(bool); !ok {
t.Errorf("expected success=true, got %v", body["success"])
}

data, ok := body["data"].(map[string]interface{})
if !ok || data["foo"] != "bar" {
t.Errorf("expected data.foo=bar, got %v", body["data"])
}
}

func TestErrorResponse(t *testing.T) {
app := fiber.New()
app.Get("/", func(c *fiber.Ctx) error {
return ErrorResponse(c, fiber.StatusBadRequest, "bad request")
})

req := httptest.NewRequest("GET", "/", nil)
resp, err := app.Test(req, -1)
if err != nil {
t.Fatalf("app.Test failed: %v", err)
}

if resp.StatusCode != fiber.StatusBadRequest {
t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
}

var body map[string]interface{}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode response failed: %v", err)
}

if ok, _ := body["success"].(bool); ok {
t.Errorf("expected success=false, got %v", body["success"])
}

if body["error"] != "bad request" {
t.Errorf("expected error message, got %v", body["error"])
}
}

func TestMessageResponse(t *testing.T) {
app := fiber.New()
app.Get("/", func(c *fiber.Ctx) error {
return MessageResponse(c, "ok")
})

req := httptest.NewRequest("GET", "/", nil)
resp, err := app.Test(req, -1)
if err != nil {
t.Fatalf("app.Test failed: %v", err)
}

var body map[string]interface{}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode response failed: %v", err)
}

if ok, _ := body["success"].(bool); !ok {
t.Errorf("expected success=true, got %v", body["success"])
}

if body["message"] != "ok" {
t.Errorf("expected message=ok, got %v", body["message"])
}
}
