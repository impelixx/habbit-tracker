package models

import (
"testing"

"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewReminderDefaults(t *testing.T) {
userID := primitive.NewObjectID()
reminder := NewReminder(userID, "08:30", "Europe/Berlin")

if reminder.UserID != userID {
t.Errorf("UserID = %v, want %v", reminder.UserID, userID)
}
if reminder.Time != "08:30" {
t.Errorf("Time = %q, want 08:30", reminder.Time)
}
if reminder.Timezone != "Europe/Berlin" {
t.Errorf("Timezone = %q, want Europe/Berlin", reminder.Timezone)
}
if !reminder.Enabled {
t.Error("Enabled should default to true")
}
if reminder.LastSent != nil {
t.Error("LastSent should default to nil")
}
if reminder.ID.IsZero() {
t.Error("ID should be initialized")
}
if reminder.CreatedAt.IsZero() || reminder.UpdatedAt.IsZero() {
t.Error("timestamps should be initialized")
}
}
