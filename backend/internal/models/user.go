package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TelegramID   int64              `bson:"telegramId" json:"telegramId"`
	Username     string             `bson:"username" json:"username"`
	FirstName    string             `bson:"firstName" json:"firstName"`
	LastName     string             `bson:"lastName" json:"lastName"`
	LanguageCode string             `bson:"languageCode" json:"languageCode"`
	Timezone     string             `bson:"timezone" json:"timezone"`
	Preferences  UserPreferences    `bson:"preferences" json:"preferences"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// UserPreferences stores user settings
type UserPreferences struct {
	ReminderEnabled bool   `bson:"reminderEnabled" json:"reminderEnabled"`
	ReminderTime    string `bson:"reminderTime" json:"reminderTime"` // HH:MM format
	Theme           string `bson:"theme" json:"theme"`               // "light" | "dark"
}

// NewUser creates a new user with default values
func NewUser(telegramID int64, username, firstName, lastName string) *User {
	return &User{
		ID:           primitive.NewObjectID(),
		TelegramID:   telegramID,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		LanguageCode: "en",
		Timezone:     "UTC",
		Preferences: UserPreferences{
			ReminderEnabled: false,
			ReminderTime:    "09:00",
			Theme:           "light",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
