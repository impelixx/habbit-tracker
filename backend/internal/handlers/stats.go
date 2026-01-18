package handlers

import (
	"context"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StatsHandler handles statistics operations
type StatsHandler struct {
	db *db.MongoDB
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(database *db.MongoDB) *StatsHandler {
	return &StatsHandler{
		db: database,
	}
}

// StatsResponse represents the statistics response
type StatsResponse struct {
	TotalTasks      int            `json:"totalTasks"`
	CompletedTasks  int            `json:"completedTasks"`
	CompletionRate  float64        `json:"completionRate"`
	Streak          int            `json:"streak"`
	LongestStreak   int            `json:"longestStreak"`
	TasksByPriority map[string]int `json:"tasksByPriority"`
	TasksBySource   map[string]int `json:"tasksBySource"`
	ThisWeek        WeekStats      `json:"thisWeek"`
	LastUpdated     time.Time      `json:"lastUpdated"`
}

// WeekStats represents weekly statistics
type WeekStats struct {
	DaysWithTasks  int `json:"daysWithTasks"`
	DaysCompleted  int `json:"daysCompleted"`
	TotalTasks     int `json:"totalTasks"`
	CompletedTasks int `json:"completedTasks"`
}

// GetStats handles GET /api/stats
func (h *StatsHandler) GetStats(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	period := c.Query("period", "all") // all, week, month, year

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Calculate date range based on period
	var startDate time.Time
	now := time.Now()

	switch period {
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	default:
		// All time - use user's creation date or a very old date
		startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	// Get all tasks in period
	tasks, err := h.db.GetTasksByUserID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tasks for stats")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get statistics")
	}

	// Filter tasks by period
	filteredTasks := make([]*models.Task, 0)
	for _, task := range tasks {
		if task.CreatedAt.After(startDate) {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Calculate statistics
	stats := h.calculateStats(filteredTasks, userID)

	return utils.SuccessResponse(c, stats)
}

// calculateStats computes statistics from tasks
func (h *StatsHandler) calculateStats(tasks []*models.Task, userID primitive.ObjectID) StatsResponse {
	stats := StatsResponse{
		TasksByPriority: make(map[string]int),
		TasksBySource:   make(map[string]int),
		LastUpdated:     time.Now(),
	}

	// Count tasks by priority and source
	for _, task := range tasks {
		stats.TotalTasks++
		if task.Completed {
			stats.CompletedTasks++
		}

		stats.TasksByPriority[task.Priority]++
		stats.TasksBySource[task.Source]++
	}

	// Calculate completion rate
	if stats.TotalTasks > 0 {
		stats.CompletionRate = float64(stats.CompletedTasks) / float64(stats.TotalTasks) * 100
	}

	// Calculate streak
	stats.Streak, stats.LongestStreak = h.calculateStreak(tasks)

	// Calculate this week stats
	stats.ThisWeek = h.calculateWeekStats(tasks)

	return stats
}

// calculateStreak calculates current and longest streak
func (h *StatsHandler) calculateStreak(tasks []*models.Task) (int, int) {
	if len(tasks) == 0 {
		return 0, 0
	}

	// Group tasks by date
	tasksByDate := make(map[string][]*models.Task)
	for _, task := range tasks {
		date := task.DueDate.Format("2006-01-02")
		tasksByDate[date] = append(tasksByDate[date], task)
	}

	// Calculate current streak (consecutive days with completed tasks)
	currentStreak := 0
	longestStreak := 0
	tempStreak := 0

	// Start from today and go backwards
	today := time.Now()
	for i := 0; i < 365; i++ { // Check up to 1 year
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		dayTasks, exists := tasksByDate[date]

		if !exists || len(dayTasks) == 0 {
			// No tasks for this day
			if i == 0 {
				// Today has no tasks, but continue checking
				continue
			}
			// Break current streak
			break
		}

		// Check if at least one task was completed
		hasCompleted := false
		for _, task := range dayTasks {
			if task.Completed {
				hasCompleted = true
				break
			}
		}

		if hasCompleted {
			if i == 0 || currentStreak > 0 || tempStreak > 0 {
				currentStreak++
				tempStreak++
				if tempStreak > longestStreak {
					longestStreak = tempStreak
				}
			}
		} else {
			// Day had tasks but none completed
			if i > 0 {
				break
			}
		}
	}

	// Find longest streak in all history
	dates := make([]string, 0, len(tasksByDate))
	for date := range tasksByDate {
		dates = append(dates, date)
	}

	// Sort dates using Go's built-in sort for better performance
	sort.Strings(dates)

	// Calculate longest streak from all dates
	tempStreak = 0
	for i, date := range dates {
		dayTasks := tasksByDate[date]
		hasCompleted := false
		for _, task := range dayTasks {
			if task.Completed {
				hasCompleted = true
				break
			}
		}

		if hasCompleted {
			tempStreak++
			if tempStreak > longestStreak {
				longestStreak = tempStreak
			}
		} else {
			tempStreak = 0
		}

		// Check for date continuity
		if i > 0 {
			prevDate, _ := time.Parse("2006-01-02", dates[i-1])
			currDate, _ := time.Parse("2006-01-02", date)
			if currDate.Sub(prevDate).Hours() > 24*1.5 {
				tempStreak = 0
				if hasCompleted {
					tempStreak = 1
				}
			}
		}
	}

	return currentStreak, longestStreak
}

// calculateWeekStats calculates statistics for the current week
func (h *StatsHandler) calculateWeekStats(tasks []*models.Task) WeekStats {
	weekStart := time.Now().AddDate(0, 0, -7)
	stats := WeekStats{}

	daysWithTasks := make(map[string]bool)
	daysCompleted := make(map[string]bool)

	for _, task := range tasks {
		if task.CreatedAt.After(weekStart) {
			stats.TotalTasks++

			date := task.DueDate.Format("2006-01-02")
			daysWithTasks[date] = true

			if task.Completed {
				stats.CompletedTasks++
				daysCompleted[date] = true
			}
		}
	}

	stats.DaysWithTasks = len(daysWithTasks)
	stats.DaysCompleted = len(daysCompleted)

	return stats
}
