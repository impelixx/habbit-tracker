package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TasksHandler handles task operations
type TasksHandler struct {
	db        *db.MongoDB
	wsHandler *WebSocketHandler
}

// NewTasksHandler creates a new tasks handler
func NewTasksHandler(database *db.MongoDB, wsHandler *WebSocketHandler) *TasksHandler {
	return &TasksHandler{
		db:        database,
		wsHandler: wsHandler,
	}
}

// CreateTaskRequest represents the create task request
type CreateTaskRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	DueDate     time.Time `json:"dueDate"`
}

// UpdateTaskRequest represents the update task request
type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Priority    *string    `json:"priority"`
	DueDate     *time.Time `json:"dueDate"`
	Completed   *bool      `json:"completed"`
}

// GetTasks handles GET /api/tasks
func (h *TasksHandler) GetTasks(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get date parameter (optional)
	dateStr := c.Query("date")
	var tasks []*models.Task
	var err error

	if dateStr != "" {
		// Parse date
		date, parseErr := time.Parse("2006-01-02", dateStr)
		if parseErr != nil {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
		}
		tasks, err = h.db.GetTasksByUserIDAndDate(ctx, userID, date)
	} else {
		// Get today's tasks by default
		tasks, err = h.db.GetTasksByUserIDAndDate(ctx, userID, time.Now())
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to get tasks")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get tasks")
	}

	return utils.SuccessResponse(c, fiber.Map{"tasks": tasks})
}

// CreateTask handles POST /api/tasks
func (h *TasksHandler) CreateTask(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	var req CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.Title == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Title is required")
	}

	// Create task
	task := models.NewTask(userID, req.Title, models.SourceWebApp)
	task.Description = req.Description

	// Set priority (default to medium if not provided)
	if req.Priority != "" {
		task.Priority = req.Priority
	}

	// Set due date (default to now if not provided)
	if !req.DueDate.IsZero() {
		task.DueDate = req.DueDate
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.db.CreateTask(ctx, task); err != nil {
		log.Error().Err(err).Msg("Failed to create task")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create task")
	}

	log.Info().
		Str("taskId", task.ID.Hex()).
		Str("userId", userID.Hex()).
		Str("title", task.Title).
		Msg("Task created via Mini App")

	// Broadcast task creation to WebSocket clients
	if h.wsHandler != nil {
		h.wsHandler.BroadcastTaskUpdate(task, "created")
	}

	return utils.SuccessResponse(c, fiber.Map{"task": task})
}

// UpdateTask handles PATCH /api/tasks/:id
func (h *TasksHandler) UpdateTask(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	taskIDStr := c.Params("id")

	// Parse task ID
	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid task ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get existing task
	task, err := h.db.GetTaskByID(ctx, taskID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get task")
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Task not found")
	}

	// Verify ownership
	if task.UserID != userID {
		return utils.ErrorResponse(c, fiber.StatusForbidden, "You don't have permission to update this task")
	}

	// Parse update request
	var req UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Update fields
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = *req.DueDate
	}
	if req.Completed != nil {
		if *req.Completed {
			task.MarkCompleted()
		} else {
			task.MarkIncomplete()
		}
	}

	// Update in database
	if err := h.db.UpdateTask(ctx, task); err != nil {
		log.Error().Err(err).Msg("Failed to update task")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to update task")
	}

	log.Info().
		Str("taskId", task.ID.Hex()).
		Bool("completed", task.Completed).
		Msg("Task updated")

	// Broadcast task update to WebSocket clients
	if h.wsHandler != nil {
		h.wsHandler.BroadcastTaskUpdate(task, "updated")
	}

	return utils.SuccessResponse(c, fiber.Map{"task": task})
}

// DeleteTask handles DELETE /api/tasks/:id
func (h *TasksHandler) DeleteTask(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	taskIDStr := c.Params("id")

	// Parse task ID
	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid task ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get existing task to verify ownership
	task, err := h.db.GetTaskByID(ctx, taskID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Task not found")
	}

	// Verify ownership
	if task.UserID != userID {
		return utils.ErrorResponse(c, fiber.StatusForbidden, "You don't have permission to delete this task")
	}

	// Delete task
	if err := h.db.DeleteTask(ctx, taskID); err != nil {
		log.Error().Err(err).Msg("Failed to delete task")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete task")
	}

	log.Info().
		Str("taskId", taskID.Hex()).
		Msg("Task deleted")

	// Broadcast task deletion to WebSocket clients
	if h.wsHandler != nil {
		h.wsHandler.BroadcastTaskUpdate(task, "deleted")
	}

	return utils.MessageResponse(c, "Task deleted successfully")
}
