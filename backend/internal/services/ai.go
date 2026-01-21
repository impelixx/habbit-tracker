package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// AIService handles AI operations using OpenRouter
type AIService struct {
	apiKey         string
	primaryModel   string
	fallbackModel  string
	httpClient     *http.Client
	rateLimiter    *RateLimiter
}

// NewAIService creates a new AI service
func NewAIService(apiKey, primaryModel, fallbackModel string) *AIService {
	return &AIService{
		apiKey:        apiKey,
		primaryModel:  primaryModel,
		fallbackModel: fallbackModel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: NewRateLimiter(50, time.Minute), // 50 requests per minute
	}
}

// ParsedTask represents a task parsed from natural language
type ParsedTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	DueTime     string `json:"dueTime"`
}

// OpenRouterRequest represents a request to OpenRouter API
type OpenRouterRequest struct {
	Model    string                   `json:"model"`
	Messages []OpenRouterMessage      `json:"messages"`
}

// OpenRouterMessage represents a message in the request
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents a response from OpenRouter API
type OpenRouterResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ParseTasksFromText uses LLM to parse tasks from natural language
func (s *AIService) ParseTasksFromText(text string) ([]*ParsedTask, error) {
	// Check rate limit
	if !s.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	}

	// Try primary model first
	tasks, err := s.parseWithModel(text, s.primaryModel)
	if err != nil {
		log.Warn().
			Err(err).
			Str("model", s.primaryModel).
			Msg("Primary model failed, trying fallback")

		// Try fallback model
		tasks, err = s.parseWithModel(text, s.fallbackModel)
		if err != nil {
			return nil, fmt.Errorf("both models failed: %w", err)
		}
	}

	return tasks, nil
}

// parseWithModel parses tasks using a specific model
func (s *AIService) parseWithModel(text, model string) ([]*ParsedTask, error) {
	prompt := s.buildPrompt(text)

	request := OpenRouterRequest{
		Model: model,
		Messages: []OpenRouterMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/impelixx/habbit-tracker")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	content := response.Choices[0].Message.Content

	// Parse JSON response
	tasks, err := s.parseTasksFromJSON(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasks from response: %w", err)
	}

	log.Info().
		Str("model", model).
		Int("taskCount", len(tasks)).
		Msg("Successfully parsed tasks with AI")

	return tasks, nil
}

// buildPrompt creates the prompt for task parsing
func (s *AIService) buildPrompt(text string) string {
	return fmt.Sprintf(`You are a task parser. Extract structured task information from the following natural language text.

Rules:
- Extract each distinct task/action mentioned
- Determine priority: "high" (urgent/important), "medium" (normal), "low" (optional/nice-to-have)
- Extract description if provided
- If time is mentioned (e.g., "7am", "evening"), include it in dueTime (HH:MM format, 24-hour)
- Return ONLY valid JSON, no other text

Input text: "%s"

Return a JSON array of tasks in this exact format:
[
  {
    "title": "Task name",
    "description": "Additional details (optional)",
    "priority": "high|medium|low",
    "dueTime": "HH:MM (optional, only if time is mentioned)"
  }
]

JSON:`, text)
}

// parseTasksFromJSON parses tasks from JSON response
func (s *AIService) parseTasksFromJSON(content string) ([]*ParsedTask, error) {
	// Try to extract JSON from markdown code blocks if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var tasks []*ParsedTask
	if err := json.Unmarshal([]byte(content), &tasks); err != nil {
		// Try parsing as single object
		var singleTask ParsedTask
		if err2 := json.Unmarshal([]byte(content), &singleTask); err2 != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		tasks = []*ParsedTask{&singleTask}
	}

	// Validate and normalize tasks
	for _, task := range tasks {
		if task.Title == "" {
			continue
		}

		// Normalize priority
		priority := strings.ToLower(task.Priority)
		switch priority {
		case "high", "medium", "low":
			task.Priority = priority
		default:
			task.Priority = models.PriorityMedium
		}
	}

	return tasks, nil
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	tokens    int
	maxTokens int
	refillAt  time.Time
	interval  time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		refillAt:  time.Now().Add(interval),
		interval:  interval,
	}
}

// Allow checks if a request is allowed
func (r *RateLimiter) Allow() bool {
	now := time.Now()

	// Refill tokens if interval has passed
	if now.After(r.refillAt) {
		r.tokens = r.maxTokens
		r.refillAt = now.Add(r.interval)
	}

	// Check if tokens are available
	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}
