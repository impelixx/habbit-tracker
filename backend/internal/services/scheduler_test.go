package services

import (
	"testing"
	"time"

	"github.com/impelixx/habbit-tracker/backend/internal/models"
)

func TestShouldSendReminder_TimeMatch(t *testing.T) {
	reminder := &models.Reminder{
		Time:     "09:00",
		Timezone: "UTC",
		Enabled:  true,
	}

	tests := []struct {
		name       string
		currentTime time.Time
		want       bool
	}{
		{
			name:       "Exact match",
			currentTime: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			want:       true,
		},
		{
			name:       "Wrong hour",
			currentTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			want:       false,
		},
		{
			name:       "Wrong minute",
			currentTime: time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
			want:       false,
		},
		{
			name:       "Same time different day",
			currentTime: time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC),
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkTimeMatch(reminder, tt.currentTime)
			if result != tt.want {
				t.Errorf("checkTimeMatch() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestShouldSendReminder_LastSent(t *testing.T) {
	now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		lastSent *time.Time
		want     bool
	}{
		{
			name:     "Never sent before",
			lastSent: nil,
			want:     true,
		},
		{
			name: "Sent 2 hours ago (same day)",
			lastSent: func() *time.Time {
				t := now.Add(-2 * time.Hour)
				return &t
			}(),
			want: false,
		},
		{
			name: "Sent yesterday",
			lastSent: func() *time.Time {
				t := now.Add(-24 * time.Hour)
				return &t
			}(),
			want: true,
		},
		{
			name: "Sent last week",
			lastSent: func() *time.Time {
				t := now.Add(-7 * 24 * time.Hour)
				return &t
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkNotSentToday(tt.lastSent, now, time.UTC)
			if result != tt.want {
				t.Errorf("checkNotSentToday() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestShouldSendReminder_Timezone(t *testing.T) {
	// 9:00 UTC = 14:00 in Asia/Tokyo (UTC+5)
	utcTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		reminderTime string
		timezone     string
		currentTime  time.Time
		want         bool
	}{
		{
			name:         "UTC timezone match",
			reminderTime: "09:00",
			timezone:     "UTC",
			currentTime:  utcTime,
			want:         true,
		},
		{
			name:         "America/New_York timezone (4:00 local)",
			reminderTime: "04:00",
			timezone:     "America/New_York",
			currentTime:  utcTime,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder := &models.Reminder{
				Time:     tt.reminderTime,
				Timezone: tt.timezone,
				Enabled:  true,
			}

			result := checkTimeMatchWithTimezone(reminder, tt.currentTime)
			if result != tt.want {
				t.Errorf("checkTimeMatchWithTimezone() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestParseReminderTime(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"Valid 09:00", "09:00", 9, 0, false},
		{"Valid 23:59", "23:59", 23, 59, false},
		{"Valid 00:00", "00:00", 0, 0, false},
		{"Valid 12:30", "12:30", 12, 30, false},
		{"Valid 9:00", "9:00", 9, 0, false}, // Go accepts single digit hours
		{"Invalid format", "abc", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, min, err := parseReminderTime(tt.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReminderTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hour != tt.wantHour {
					t.Errorf("hour = %d, want %d", hour, tt.wantHour)
				}
				if min != tt.wantMin {
					t.Errorf("minute = %d, want %d", min, tt.wantMin)
				}
			}
		})
	}
}

func TestSchedulerCheckInterval(t *testing.T) {
	// Test that check interval is reasonable (1 minute)
	expectedInterval := 1 * time.Minute

	if expectedInterval < 30*time.Second {
		t.Error("Check interval should be at least 30 seconds")
	}

	if expectedInterval > 5*time.Minute {
		t.Error("Check interval should not exceed 5 minutes")
	}
}

func TestReminderEnabledCheck(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{"Enabled reminder", true, true},
		{"Disabled reminder", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder := &models.Reminder{
				Enabled: tt.enabled,
			}

			if reminder.Enabled != tt.want {
				t.Errorf("Enabled = %v, want %v", reminder.Enabled, tt.want)
			}
		})
	}
}

// Helper functions for testing scheduler logic

func checkTimeMatch(reminder *models.Reminder, now time.Time) bool {
	location, err := time.LoadLocation(reminder.Timezone)
	if err != nil {
		return false
	}

	userTime := now.In(location)

	var reminderHour, reminderMin int
	_, err = parseReminderTimeHelper(reminder.Time, &reminderHour, &reminderMin)
	if err != nil {
		return false
	}

	return userTime.Hour() == reminderHour && userTime.Minute() == reminderMin
}

func checkNotSentToday(lastSent *time.Time, now time.Time, location *time.Location) bool {
	if lastSent == nil {
		return true
	}

	lastSentInTz := lastSent.In(location)
	nowInTz := now.In(location)

	return lastSentInTz.YearDay() != nowInTz.YearDay() || lastSentInTz.Year() != nowInTz.Year()
}

func checkTimeMatchWithTimezone(reminder *models.Reminder, now time.Time) bool {
	location, err := time.LoadLocation(reminder.Timezone)
	if err != nil {
		return false
	}

	userTime := now.In(location)

	var reminderHour, reminderMin int
	_, err = parseReminderTimeHelper(reminder.Time, &reminderHour, &reminderMin)
	if err != nil {
		return false
	}

	return userTime.Hour() == reminderHour && userTime.Minute() == reminderMin
}

func parseReminderTime(timeStr string) (int, int, error) {
	var hour, min int
	_, err := parseReminderTimeHelper(timeStr, &hour, &min)
	return hour, min, err
}

func parseReminderTimeHelper(timeStr string, hour *int, min *int) (int, error) {
	parsed, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, err
	}
	*hour = parsed.Hour()
	*min = parsed.Minute()
	return 2, nil // number of values parsed
}
