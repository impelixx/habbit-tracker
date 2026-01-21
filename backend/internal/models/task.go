package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Priority levels for tasks
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

// Source indicates where the task was created
const (
	SourceBot    = "bot"
	SourceWebApp = "webapp"
	SourceVoice  = "voice"
)

// Task represents a task or habit
type Task struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Priority    string             `bson:"priority" json:"priority"` // low, medium, high
	DueDate     time.Time          `bson:"dueDate" json:"dueDate"`
	Completed   bool               `bson:"completed" json:"completed"`
	CompletedAt *time.Time         `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	Source      string             `bson:"source" json:"source"` // bot, webapp, voice
	Metadata    *TaskMetadata      `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// TaskMetadata stores additional information about the task
type TaskMetadata struct {
	Transcription string `bson:"transcription,omitempty" json:"transcription,omitempty"`
	AIModel       string `bson:"aiModel,omitempty" json:"aiModel,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(userID primitive.ObjectID, title string, source string) *Task {
	return &Task{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Title:       title,
		Description: "",
		Priority:    PriorityMedium,
		DueDate:     time.Now(),
		Completed:   false,
		CompletedAt: nil,
		Source:      source,
		Metadata:    nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// MarkCompleted marks the task as completed
func (t *Task) MarkCompleted() {
	t.Completed = true
	now := time.Now()
	t.CompletedAt = &now
	t.UpdatedAt = now
}

// MarkIncomplete marks the task as incomplete
func (t *Task) MarkIncomplete() {
	t.Completed = false
	t.CompletedAt = nil
	t.UpdatedAt = time.Now()
}

// ParseObjectID parses a hex string into an ObjectID
func ParseObjectID(hex string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(hex)
}
