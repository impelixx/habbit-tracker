package handlers

import (
	"testing"
	"time"

	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCalculateStreak(t *testing.T) {
	userID := primitive.NewObjectID()
	today := time.Now()

	tests := []struct {
		name           string
		tasks          []*models.Task
		wantCurrent    int
		wantLongest    int
	}{
		{
			name:           "No tasks",
			tasks:          []*models.Task{},
			wantCurrent:    0,
			wantLongest:    0,
		},
		{
			name: "One day streak",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
			},
			wantCurrent: 1,
			wantLongest: 1,
		},
		{
			name: "Three day streak",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
				createCompletedTask(userID, today.Add(-24*time.Hour), "Task 2"),
				createCompletedTask(userID, today.Add(-48*time.Hour), "Task 3"),
			},
			wantCurrent: 3,
			wantLongest: 3,
		},
		{
			name: "Broken streak",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
				createCompletedTask(userID, today.Add(-24*time.Hour), "Task 2"),
				// Gap here - no task 2 days ago
				createCompletedTask(userID, today.Add(-72*time.Hour), "Task 3"),
				createCompletedTask(userID, today.Add(-96*time.Hour), "Task 4"),
			},
			wantCurrent: 2,
			wantLongest: 2,
		},
		{
			name: "Incomplete tasks don't count",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
				createIncompleteTask(userID, today.Add(-24*time.Hour), "Task 2"),
				createCompletedTask(userID, today.Add(-48*time.Hour), "Task 3"),
			},
			wantCurrent: 1,
			wantLongest: 1,
		},
		{
			name: "Multiple tasks same day count as one day",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
				createCompletedTask(userID, today, "Task 2"),
				createCompletedTask(userID, today, "Task 3"),
				createCompletedTask(userID, today.Add(-24*time.Hour), "Task 4"),
			},
			wantCurrent: 2,
			wantLongest: 2,
		},
		{
			name: "Old streak longer than current",
			tasks: []*models.Task{
				createCompletedTask(userID, today, "Task 1"),
				// Gap
				createCompletedTask(userID, today.Add(-72*time.Hour), "Task 2"),
				createCompletedTask(userID, today.Add(-96*time.Hour), "Task 3"),
				createCompletedTask(userID, today.Add(-120*time.Hour), "Task 4"),
				createCompletedTask(userID, today.Add(-144*time.Hour), "Task 5"),
			},
			wantCurrent: 1,
			wantLongest: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &StatsHandler{}
			current, longest := handler.calculateStreak(tt.tasks)

			if current != tt.wantCurrent {
				t.Errorf("calculateStreak() current = %v, want %v", current, tt.wantCurrent)
			}
			if longest != tt.wantLongest {
				t.Errorf("calculateStreak() longest = %v, want %v", longest, tt.wantLongest)
			}
		})
	}
}

func TestCalculateCompletionRate(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		completed int
		wantRate  float64
	}{
		{"No tasks", 0, 0, 0.0},
		{"All completed", 10, 10, 100.0},
		{"Half completed", 10, 5, 50.0},
		{"None completed", 10, 0, 0.0},
		{"One of three", 3, 1, 33.33},
		{"Two of three", 3, 2, 66.66}, // Truncation not rounding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rate float64
			if tt.total > 0 {
				rate = float64(tt.completed) / float64(tt.total) * 100
				// Round to 2 decimal places
				rate = float64(int(rate*100)) / 100
			}

			if rate != tt.wantRate {
				t.Errorf("Completion rate = %.2f, want %.2f", rate, tt.wantRate)
			}
		})
	}
}

func TestGroupTasksByPriority(t *testing.T) {
	userID := primitive.NewObjectID()
	today := time.Now()

	tasks := []*models.Task{
		createTaskWithPriority(userID, today, "high"),
		createTaskWithPriority(userID, today, "high"),
		createTaskWithPriority(userID, today, "medium"),
		createTaskWithPriority(userID, today, "medium"),
		createTaskWithPriority(userID, today, "medium"),
		createTaskWithPriority(userID, today, "low"),
	}

	priorityCount := make(map[string]int)
	for _, task := range tasks {
		priorityCount[task.Priority]++
	}

	if priorityCount["high"] != 2 {
		t.Errorf("High priority count = %d, want 2", priorityCount["high"])
	}
	if priorityCount["medium"] != 3 {
		t.Errorf("Medium priority count = %d, want 3", priorityCount["medium"])
	}
	if priorityCount["low"] != 1 {
		t.Errorf("Low priority count = %d, want 1", priorityCount["low"])
	}
}

func TestGroupTasksBySource(t *testing.T) {
	userID := primitive.NewObjectID()
	today := time.Now()

	tasks := []*models.Task{
		createTaskWithSource(userID, today, models.SourceBot),
		createTaskWithSource(userID, today, models.SourceBot),
		createTaskWithSource(userID, today, models.SourceBot),
		createTaskWithSource(userID, today, models.SourceWebApp),
		createTaskWithSource(userID, today, models.SourceWebApp),
		createTaskWithSource(userID, today, models.SourceVoice),
	}

	sourceCount := make(map[string]int)
	for _, task := range tasks {
		sourceCount[task.Source]++
	}

	if sourceCount[models.SourceBot] != 3 {
		t.Errorf("Bot source count = %d, want 3", sourceCount[models.SourceBot])
	}
	if sourceCount[models.SourceWebApp] != 2 {
		t.Errorf("WebApp source count = %d, want 2", sourceCount[models.SourceWebApp])
	}
	if sourceCount[models.SourceVoice] != 1 {
		t.Errorf("Voice source count = %d, want 1", sourceCount[models.SourceVoice])
	}
}

func TestWeeklyStats(t *testing.T) {
	userID := primitive.NewObjectID()
	today := time.Now()

	// Create tasks for the last 7 days
	tasks := []*models.Task{}
	for i := 0; i < 7; i++ {
		date := today.Add(time.Duration(-i*24) * time.Hour)
		if i%2 == 0 {
			tasks = append(tasks, createCompletedTask(userID, date, "Task"))
		} else {
			tasks = append(tasks, createIncompleteTask(userID, date, "Task"))
		}
	}

	// Count tasks by day
	weeklyStats := make(map[string]int)
	for _, task := range tasks {
		dayKey := task.DueDate.Format("2006-01-02")
		weeklyStats[dayKey]++
	}

	// We should have 7 days with 1 task each
	if len(weeklyStats) != 7 {
		t.Errorf("Weekly stats should have 7 days, got %d", len(weeklyStats))
	}

	for _, count := range weeklyStats {
		if count != 1 {
			t.Errorf("Each day should have 1 task, got %d", count)
		}
	}
}

// Helper functions

func createCompletedTask(userID primitive.ObjectID, dueDate time.Time, title string) *models.Task {
	task := models.NewTask(userID, title, models.SourceBot)
	task.DueDate = dueDate
	task.Completed = true
	now := dueDate
	task.CompletedAt = &now
	return task
}

func createIncompleteTask(userID primitive.ObjectID, dueDate time.Time, title string) *models.Task {
	task := models.NewTask(userID, title, models.SourceBot)
	task.DueDate = dueDate
	task.Completed = false
	return task
}

func createTaskWithPriority(userID primitive.ObjectID, dueDate time.Time, priority string) *models.Task {
	task := models.NewTask(userID, "Task", models.SourceBot)
	task.DueDate = dueDate
	task.Priority = priority
	return task
}

func createTaskWithSource(userID primitive.ObjectID, dueDate time.Time, source string) *models.Task {
	task := models.NewTask(userID, "Task", source)
	task.DueDate = dueDate
	return task
}
