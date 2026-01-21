package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewTask(t *testing.T) {
	userID := primitive.NewObjectID()
	title := "Test Task"
	source := SourceBot

	task := NewTask(userID, title, source)

	if task.UserID != userID {
		t.Errorf("Expected UserID %v, got %v", userID, task.UserID)
	}

	if task.Title != title {
		t.Errorf("Expected Title %s, got %s", title, task.Title)
	}

	if task.Source != source {
		t.Errorf("Expected Source %s, got %s", source, task.Source)
	}

	if task.Priority != PriorityMedium {
		t.Errorf("Expected Priority %s, got %s", PriorityMedium, task.Priority)
	}

	if task.Completed {
		t.Error("Expected Completed to be false")
	}

	if task.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil")
	}
}

func TestMarkCompleted(t *testing.T) {
	task := NewTask(primitive.NewObjectID(), "Test", SourceBot)

	if task.Completed {
		t.Error("Task should not be completed initially")
	}

	task.MarkCompleted()

	if !task.Completed {
		t.Error("Task should be marked as completed")
	}

	if task.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}

	// Check if CompletedAt is recent (within last second)
	if time.Since(*task.CompletedAt) > time.Second {
		t.Error("CompletedAt should be recent")
	}
}

func TestMarkIncomplete(t *testing.T) {
	task := NewTask(primitive.NewObjectID(), "Test", SourceBot)
	task.MarkCompleted()

	if !task.Completed {
		t.Error("Task should be completed")
	}

	task.MarkIncomplete()

	if task.Completed {
		t.Error("Task should be marked as incomplete")
	}

	if task.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
}

func TestParseObjectID(t *testing.T) {
	id := primitive.NewObjectID()
	hex := id.Hex()

	parsed, err := ParseObjectID(hex)
	if err != nil {
		t.Errorf("Failed to parse ObjectID: %v", err)
	}

	if parsed != id {
		t.Errorf("Expected %v, got %v", id, parsed)
	}

	// Test invalid hex
	_, err = ParseObjectID("invalid")
	if err == nil {
		t.Error("Expected error for invalid ObjectID")
	}
}
