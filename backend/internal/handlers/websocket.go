package handlers

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/impelixx/habbit-tracker/backend/internal/services"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebSocketHandler manages WebSocket connections
type WebSocketHandler struct {
	db          *db.MongoDB
	authService *services.AuthService
	hub         *Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(database *db.MongoDB, authService *services.AuthService) *WebSocketHandler {
	handler := &WebSocketHandler{
		db:          database,
		authService: authService,
		hub:         NewHub(),
	}

	// Start the hub
	go handler.hub.Run()

	return handler
}

// HandleUpgrade upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) HandleUpgrade(c *fiber.Ctx) error {
	// Check if this is a WebSocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// HandleConnection handles WebSocket connections
func (h *WebSocketHandler) HandleConnection(c *websocket.Conn) {
	// Get user ID from JWT token
	token := c.Query("token")
	if token == "" {
		log.Warn().Msg("WebSocket connection without token")
		c.Close()
		return
	}

	// Validate token
	claims, err := h.authService.ValidateJWT(token)
	if err != nil {
		log.Error().Err(err).Msg("Invalid WebSocket token")
		c.Close()
		return
	}

	// Convert string UserID to ObjectID
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("Invalid user ID in token")
		c.Close()
		return
	}

	// Create client
	client := &Client{
		hub:    h.hub,
		conn:   c,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	// Register client
	h.hub.register <- client

	log.Info().
		Str("userId", userID.Hex()).
		Msg("WebSocket client connected")

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastTaskUpdate broadcasts task update to all connected clients
func (h *WebSocketHandler) BroadcastTaskUpdate(task *models.Task, action string) {
	message := WSMessage{
		Type:   "task_update",
		Action: action, // created, updated, deleted
		Data:   task,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal WebSocket message")
		return
	}

	// Send to all clients of this user
	h.hub.BroadcastToUser(task.UserID, data)
}

// BroadcastTasksUpdate broadcasts multiple tasks update
func (h *WebSocketHandler) BroadcastTasksUpdate(userID primitive.ObjectID, tasks []*models.Task, action string) {
	message := WSMessage{
		Type:   "tasks_batch",
		Action: action,
		Data:   tasks,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal WebSocket message")
		return
	}

	h.hub.BroadcastToUser(userID, data)
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// Hub maintains active WebSocket clients
type Hub struct {
	// Registered clients by user ID
	clients map[primitive.ObjectID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[primitive.ObjectID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()

			log.Debug().
				Str("userId", client.userID.Hex()).
				Int("clientCount", len(h.clients[client.userID])).
				Msg("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)

					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()

			log.Debug().
				Str("userId", client.userID.Hex()).
				Msg("Client unregistered")
		}
	}
}

// BroadcastToUser sends message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID primitive.ObjectID, message []byte) {
	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Client's send buffer is full, close it
			h.unregister <- client
		}
	}
}

// GetClientCount returns number of connected clients for a user
func (h *Hub) GetClientCount(userID primitive.ObjectID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

// Client represents a WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID primitive.ObjectID
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// readPump reads messages from WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		// Handle incoming messages (currently we only send from server to client)
		log.Debug().
			Str("userId", c.userID.Hex()).
			Str("message", string(message)).
			Msg("Received WebSocket message")
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error().Err(err).Msg("Failed to write WebSocket message")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetHub returns the WebSocket hub for broadcasting
func (h *WebSocketHandler) GetHub() *Hub {
	return h.hub
}
