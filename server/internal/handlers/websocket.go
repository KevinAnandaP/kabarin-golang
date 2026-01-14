package handlers

import (
	"log"

	ws "ngabarin/server/internal/websocket"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var (
	// WSHub is the global WebSocket hub instance
	WSHub *ws.Hub
)

// InitWebSocket initializes the WebSocket hub
func InitWebSocket() {
	WSHub = ws.NewHub()
	go WSHub.Run()
	log.Println("✅ WebSocket Hub initialized")
}

// WebSocketUpgrade checks if the request should be upgraded to WebSocket
func WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if this is a WebSocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}

	return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
		"success": false,
		"error":   "WebSocket upgrade required",
	})
}

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(c *websocket.Conn) {
	// Get user info from context (set by auth middleware)
	userID := c.Locals("userID").(string)
	uniqueID := c.Locals("uniqueID").(string)

	// Create new client
	client := ws.NewClient(userID, uniqueID, c, WSHub)

	// Register client
	WSHub.Register <- client

	// Start read and write pumps in separate goroutines
	go client.WritePump()
	client.ReadPump() // This blocks until connection closes
}

// GetWebSocketStats returns WebSocket connection statistics
func GetWebSocketStats(c *fiber.Ctx) error {
	if WSHub == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"error":   "WebSocket hub not initialized",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"onlineUsers": WSHub.GetOnlineCount(),
			"userIds":     WSHub.GetOnlineUsers(),
		},
	})
}
