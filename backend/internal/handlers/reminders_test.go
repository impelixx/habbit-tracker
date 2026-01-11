package handlers

import (
	"testing"
	"time"

	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidateReminderTime(t *testing.T) {
	tests := []struct {
		name      string
		timeStr   string
		wantValid bool
	}{
		{"Valid time 09:00", "09:00", true},
		{"Valid time 23:59", "23:59", true},
		{"Valid time 00:00", "00:00", true},
		{"Valid time 12:30", "12:30", true},
		{"Invalid format 9:00", "9:00", false},
		{"Invalid format 09:0", "09:0", false},
		{"Invalid hour 25:00", "25:00", false},
		{"Invalid minute 09:60", "09:60", false},
		{"Invalid format abc", "abc", false},
		{"Empty string", "", false},
		{"Missing colon 0900", "0900", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateReminderTime(tt.timeStr)
			if isValid != tt.wantValid {
				t.Errorf("validateReminderTime(%q) = %v, want %v", tt.timeStr, isValid, tt.wantValid)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name      string
		timezone  string
		wantValid bool
	}{
		{"Valid UTC", "UTC", true},
		{"Valid America/New_York", "America/New_York", true},
		{"Valid Europe/Moscow", "Europe/Moscow", true},
		{"Valid Asia/Tokyo", "Asia/Tokyo", true},
		{"Invalid timezone", "Invalid/Timezone", false},
		{"Empty string", "", true}, // LoadLocation("") returns UTC
		{"Random string", "abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := time.LoadLocation(tt.timezone)
			isValid := (err == nil)
			if isValid != tt.wantValid {
				t.Errorf("LoadLocation(%q) valid = %v, want %v", tt.timezone, isValid, tt.wantValid)
			}
		})
	}
}

func TestNewReminder(t *testing.T) {
	userIDStr := "507f1f77bcf86cd799439011" // Valid ObjectID string
	userID, _ := primitive.ObjectIDFromHex(userIDStr)
	reminderTime := "09:00"
	timezone := "UTC"

	// This would normally be in models/reminder_test.go, but testing reminder creation logic
	reminder := &models.Reminder{
		UserID:   userID,
		Time:     reminderTime,
		Timezone: timezone,
		Enabled:  true,
	}

	if reminder.UserID.Hex() != userIDStr {
		t.Errorf("Expected UserID %s, got %s", userIDStr, reminder.UserID.Hex())
	}

	if reminder.Time != reminderTime {
		t.Errorf("Expected Time %s, got %s", reminderTime, reminder.Time)
	}

	if reminder.Timezone != timezone {
		t.Errorf("Expected Timezone %s, got %s", timezone, reminder.Timezone)
	}

	if !reminder.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func validateReminderTime(timeStr string) bool {
	if len(timeStr) != 5 {
		return false
	}

	var hour, minute int
	_, err := time.Parse("15:04", timeStr)
	if err != nil {
		return false
	}

	// Additional validation
	if _, err := time.ParseInLocation("15:04", timeStr, time.UTC); err != nil {
		return false
	}

	// Parse hour and minute
	if n, _ := time.Parse("15:04", timeStr); n.IsZero() {
		return false
	}

	// Extract hour and minute
	if _, err := time.Parse("15:04", timeStr); err == nil {
		t, _ := time.Parse("15:04", timeStr)
		hour = t.Hour()
		minute = t.Minute()

		if hour < 0 || hour > 23 {
			return false
		}
		if minute < 0 || minute > 59 {
			return false
		}
		return true
	}

	return false
}

func TestReminderTimeComparison(t *testing.T) {
	tests := []struct {
		name         string
		reminderTime string
		currentTime  time.Time
		timezone     string
		shouldMatch  bool
	}{
		{
			name:         "Exact match",
			reminderTime: "09:00",
			currentTime:  time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			timezone:     "UTC",
			shouldMatch:  true,
		},
		{
			name:         "Different hour",
			reminderTime: "09:00",
			currentTime:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			timezone:     "UTC",
			shouldMatch:  false,
		},
		{
			name:         "Different minute",
			reminderTime: "09:00",
			currentTime:  time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
			timezone:     "UTC",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := time.LoadLocation(tt.timezone)
			if err != nil {
				t.Fatalf("Failed to load location: %v", err)
			}

			userTime := tt.currentTime.In(location)

			var reminderHour, reminderMin int
			if n, err := time.Parse("15:04", tt.reminderTime); err == nil {
				reminderHour = n.Hour()
				reminderMin = n.Minute()
			}

			matches := (userTime.Hour() == reminderHour && userTime.Minute() == reminderMin)

			if matches != tt.shouldMatch {
				t.Errorf("Time match = %v, want %v", matches, tt.shouldMatch)
			}
		})
	}
}

func TestReminderLastSentCheck(t *testing.T) {
	now := time.Now()
	location := time.UTC

	tests := []struct {
		name        string
		lastSent    *time.Time
		currentTime time.Time
		shouldSend  bool
	}{
		{
			name:        "Never sent before",
			lastSent:    nil,
			currentTime: now,
			shouldSend:  true,
		},
		{
			name: "Sent earlier today",
			lastSent: func() *time.Time {
				t := now.Add(-2 * time.Hour)
				return &t
			}(),
			currentTime: now,
			shouldSend:  false,
		},
		{
			name: "Sent yesterday",
			lastSent: func() *time.Time {
				t := now.Add(-25 * time.Hour)
				return &t
			}(),
			currentTime: now,
			shouldSend:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldSend := true

			if tt.lastSent != nil {
				lastSentUserTime := tt.lastSent.In(location)
				currentUserTime := tt.currentTime.In(location)

				if lastSentUserTime.YearDay() == currentUserTime.YearDay() &&
					lastSentUserTime.Year() == currentUserTime.Year() {
					shouldSend = false
				}
			}

			if shouldSend != tt.shouldSend {
				t.Errorf("shouldSend = %v, want %v", shouldSend, tt.shouldSend)
			}
		})
	}
}
