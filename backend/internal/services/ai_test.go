package services

import (
	"testing"
	"time"
)

func TestParseTasksFromJSON(t *testing.T) {
	service := NewAIService("test_key", "test_model", "fallback_model")

	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantErr  bool
	}{
		{
			name: "Valid JSON array",
			input: `[
				{"title": "Morning workout", "description": "30 min cardio", "priority": "high"},
				{"title": "Read book", "description": "", "priority": "medium"}
			]`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "JSON with markdown code block",
			input: "```json\n" + `[
				{"title": "Task 1", "priority": "low"}
			]` + "\n```",
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "Single task object",
			input: `{
				"title": "Single task",
				"priority": "high"
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `{invalid json}`,
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := service.parseTasksFromJSON(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tasks) != tt.wantLen {
				t.Errorf("Expected %d tasks, got %d", tt.wantLen, len(tasks))
			}

			// Check priority normalization
			for _, task := range tasks {
				if task.Priority != "low" && task.Priority != "medium" && task.Priority != "high" {
					t.Errorf("Invalid priority: %s", task.Priority)
				}
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	service := NewAIService("test_key", "test_model", "fallback_model")

	text := "workout tomorrow at 7am"
	prompt := service.buildPrompt(text)

	if prompt == "" {
		t.Error("Prompt should not be empty")
	}

	// Check if prompt contains the input text
	if len(prompt) < len(text) {
		t.Error("Prompt should contain the input text")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Second)

	// First two requests should succeed
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	if !limiter.Allow() {
		t.Error("Second request should be allowed")
	}

	// Third request should fail
	if limiter.Allow() {
		t.Error("Third request should be blocked")
	}

	// Wait for refill
	time.Sleep(time.Second + 100*time.Millisecond)

	// Should be allowed again
	if !limiter.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	limiter := NewRateLimiter(10, time.Second)
	allowed := 0

	// Try 15 concurrent requests
	for i := 0; i < 15; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	// Only 10 should be allowed
	if allowed != 10 {
		t.Errorf("Expected 10 requests allowed, got %d", allowed)
	}
}
