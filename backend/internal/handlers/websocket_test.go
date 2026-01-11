package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		// Stop hub by closing channels
		close(hub.register)
		close(hub.unregister)
	}()

	userID := primitive.NewObjectID()
	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // Wait for registration

	// Check client count
	count := hub.GetClientCount(userID)
	if count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	userID := primitive.NewObjectID()
	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Unregister client
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	// Check client count
	count := hub.GetClientCount(userID)
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestHub_MultipleClientsPerUser(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	userID := primitive.NewObjectID()

	// Register 3 clients for the same user
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			hub:    hub,
			send:   make(chan []byte, 256),
			userID: userID,
		}
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	// Check client count
	count := hub.GetClientCount(userID)
	if count != 3 {
		t.Errorf("Expected 3 clients, got %d", count)
	}

	// Unregister one client
	hub.unregister <- clients[0]
	time.Sleep(10 * time.Millisecond)

	// Check client count again
	count = hub.GetClientCount(userID)
	if count != 2 {
		t.Errorf("Expected 2 clients after unregister, got %d", count)
	}
}

func TestHub_BroadcastToUser(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	userID := primitive.NewObjectID()
	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	message := []byte(`{"type":"test","data":"hello"}`)
	hub.BroadcastToUser(userID, message)

	// Check if message was received
	select {
	case msg := <-client.send:
		if string(msg) != string(message) {
			t.Errorf("Expected message %s, got %s", message, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for message")
	}
}

func TestHub_BroadcastToNonExistentUser(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	userID := primitive.NewObjectID()

	// Broadcast to user with no connections (should not panic)
	message := []byte(`{"type":"test"}`)
	hub.BroadcastToUser(userID, message)

	// If we reach here without panic, test passes
	time.Sleep(10 * time.Millisecond)
}

func TestHub_MultipleUsers(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	user1ID := primitive.NewObjectID()
	user2ID := primitive.NewObjectID()

	client1 := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: user1ID,
	}
	client2 := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: user2ID,
	}

	// Register both clients
	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast to user1
	message := []byte(`{"type":"test","user":"1"}`)
	hub.BroadcastToUser(user1ID, message)

	// Check that only client1 received the message
	select {
	case msg := <-client1.send:
		if string(msg) != string(message) {
			t.Errorf("Client1: Expected message %s, got %s", message, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client1: Timeout waiting for message")
	}

	// Check that client2 did not receive the message
	select {
	case msg := <-client2.send:
		t.Errorf("Client2: Should not receive message, but got %s", msg)
	case <-time.After(50 * time.Millisecond):
		// Expected timeout, test passes
	}
}

func TestHub_ClientCountForDifferentUsers(t *testing.T) {
	hub := NewHub()

	// Start hub
	go hub.Run()
	defer func() {
		close(hub.register)
		close(hub.unregister)
	}()

	user1ID := primitive.NewObjectID()
	user2ID := primitive.NewObjectID()

	// Register 2 clients for user1
	for i := 0; i < 2; i++ {
		client := &Client{
			hub:    hub,
			send:   make(chan []byte, 256),
			userID: user1ID,
		}
		hub.register <- client
	}

	// Register 3 clients for user2
	for i := 0; i < 3; i++ {
		client := &Client{
			hub:    hub,
			send:   make(chan []byte, 256),
			userID: user2ID,
		}
		hub.register <- client
	}

	time.Sleep(10 * time.Millisecond)

	// Check client counts
	count1 := hub.GetClientCount(user1ID)
	if count1 != 2 {
		t.Errorf("User1: Expected 2 clients, got %d", count1)
	}

	count2 := hub.GetClientCount(user2ID)
	if count2 != 3 {
		t.Errorf("User2: Expected 3 clients, got %d", count2)
	}
}
