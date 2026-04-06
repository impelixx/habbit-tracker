package models

import "testing"

func TestNewUserDefaults(t *testing.T) {
user := NewUser(123, "john", "John", "Doe")

if user.TelegramID != 123 {
t.Errorf("TelegramID = %d, want 123", user.TelegramID)
}
if user.Username != "john" {
t.Errorf("Username = %q, want john", user.Username)
}
if user.FirstName != "John" || user.LastName != "Doe" {
t.Errorf("unexpected names: %s %s", user.FirstName, user.LastName)
}
if user.LanguageCode != "en" {
t.Errorf("LanguageCode = %q, want en", user.LanguageCode)
}
if user.Timezone != "UTC" {
t.Errorf("Timezone = %q, want UTC", user.Timezone)
}
if user.Preferences.ReminderEnabled {
t.Error("ReminderEnabled should default to false")
}
if user.Preferences.ReminderTime != "09:00" {
t.Errorf("ReminderTime = %q, want 09:00", user.Preferences.ReminderTime)
}
if user.Preferences.Theme != "light" {
t.Errorf("Theme = %q, want light", user.Preferences.Theme)
}
if user.ID.IsZero() {
t.Error("ID should be initialized")
}
if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
t.Error("timestamps should be initialized")
}
}
