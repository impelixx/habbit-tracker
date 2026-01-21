package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Reminder represents a scheduled reminder for a user
type Reminder struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Time      string             `bson:"time" json:"time"`         // HH:MM format
	Timezone  string             `bson:"timezone" json:"timezone"` // e.g., "America/New_York"
	Enabled   bool               `bson:"enabled" json:"enabled"`
	LastSent  *time.Time         `bson:"lastSent,omitempty" json:"lastSent,omitempty"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// NewReminder creates a new reminder with default values
func NewReminder(userID primitive.ObjectID, time, timezone string) *Reminder {
	return &Reminder{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Time:      time,
		Timezone:  timezone,
		Enabled:   true,
		LastSent:  nil,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
