package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/config"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/handlers"
	"github.com/impelixx/habbit-tracker/backend/internal/middleware"
	"github.com/impelixx/habbit-tracker/backend/internal/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	log.Info().Msg("Starting Habit Tracker Bot...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to MongoDB
	log.Info().Msg("Connecting to MongoDB...")
	database, err := db.Connect(cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer database.Close()
	log.Info().Msg("Connected to MongoDB successfully")

	// Initialize AI service (optional)
	var aiService *services.AIService
	if cfg.OpenRouterAPIKey != "" {
		log.Info().Msg("Initializing AI service...")
		aiService = services.NewAIService(
			cfg.OpenRouterAPIKey,
			cfg.OpenRouterLLMModel,
			cfg.OpenRouterLLMFallback,
		)
		log.Info().Msg("AI service initialized")
	} else {
		log.Warn().Msg("OpenRouter API key not provided. AI features will be disabled.")
	}

	// Initialize Telegram service
	log.Info().Msg("Initializing Telegram bot...")
	telegramService, err := services.NewTelegramService(cfg.TelegramBotToken, database, aiService)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Telegram service")
	}

	// Set webhook if URL is provided
	if cfg.TelegramWebhookURL != "" {
		log.Info().Msg("Setting Telegram webhook...")
		if err := telegramService.SetWebhook(cfg.TelegramWebhookURL); err != nil {
			log.Fatal().Err(err).Msg("Failed to set webhook")
		}
	} else {
		log.Warn().Msg("No webhook URL provided. Bot will not receive updates.")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Habit Tracker API v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} - ${method} ${path} (${latency})\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins[0], // TODO: Support multiple origins
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE, OPTIONS",
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Initialize services
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpiration, cfg.TelegramBotToken)

	// Initialize reminder scheduler
	schedulerService := services.NewSchedulerService(database, telegramService.GetBot())
	schedulerService.Start()
	defer schedulerService.Stop()
	log.Info().Msg("Reminder scheduler initialized")

	// Initialize handlers
	webhookHandler := handlers.NewWebhookHandler(telegramService)
	authHandler := handlers.NewAuthHandler(authService, database)
	wsHandler := handlers.NewWebSocketHandler(database, authService)
	tasksHandler := handlers.NewTasksHandler(database, wsHandler)
	remindersHandler := handlers.NewRemindersHandler(database)
	statsHandler := handlers.NewStatsHandler(database)

	// Set broadcaster for telegram service to enable WebSocket updates
	telegramService.SetBroadcaster(wsHandler)

	// Routes
	api := app.Group("/api")

	// Telegram webhook
	api.Post("/webhook/telegram", webhookHandler.HandleWebhook)

	// Auth routes (no auth required)
	api.Post("/auth/verify", authHandler.Verify)

	// Protected routes (require JWT auth)
	protected := api.Group("", middleware.AuthMiddleware(authService))
	protected.Get("/auth/me", authHandler.GetMe)
	protected.Get("/tasks", tasksHandler.GetTasks)
	protected.Post("/tasks", tasksHandler.CreateTask)
	protected.Patch("/tasks/:id", tasksHandler.UpdateTask)
	protected.Delete("/tasks/:id", tasksHandler.DeleteTask)

	// Statistics
	protected.Get("/stats", statsHandler.GetStats)

	// Reminders
	protected.Get("/reminders", remindersHandler.GetReminder)
	protected.Post("/reminders/set", remindersHandler.SetReminder)
	protected.Delete("/reminders", remindersHandler.DeleteReminder)

	// WebSocket route
	api.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	api.Get("/ws", websocket.New(func(c *websocket.Conn) {
		wsHandler.HandleConnection(c)
	}))

	// Start server in a goroutine
	go func() {
		addr := ":" + cfg.Port
		log.Info().Str("addr", addr).Msg("Starting server...")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}
