package routes

import (
	"ngabarin/server/internal/handlers"
	"ngabarin/server/internal/middleware"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// InitWebSocket initializes the WebSocket hub
func InitWebSocket() {
	handlers.InitWebSocket()
}

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App) {
	// API v1 group
	api := app.Group("/api/v1")

	// Health check (public)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Ngabarin API is running",
		})
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", middleware.StrictRateLimiter(), handlers.Register)
	auth.Post("/login", middleware.StrictRateLimiter(), handlers.Login)
	auth.Post("/refresh", middleware.StrictRateLimiter(), handlers.RefreshToken)
	auth.Post("/logout", middleware.AuthMiddleware, handlers.Logout)
	auth.Get("/me", middleware.AuthMiddleware, handlers.GetMe)
	auth.Get("/google", handlers.GoogleOAuthURL)
	auth.Get("/google/callback", handlers.GoogleOAuthCallback)
	// Contact routes (protected)
	contacts := api.Group("/contacts", middleware.AuthMiddleware)
	contacts.Post("/", handlers.AddContact)
	contacts.Get("/", handlers.GetContacts)
	contacts.Get("/search", handlers.SearchContacts)
	contacts.Delete("/:contactId", handlers.RemoveContact)

	// Message routes (protected)
	messages := api.Group("/messages", middleware.AuthMiddleware)
	messages.Get("/chats", handlers.GetChats) // Get all chats (contacts + non-contacts with messages)
	messages.Post("/", handlers.SendMessage)
	messages.Get("/:chatId", handlers.GetMessages)
	messages.Put("/read", handlers.MarkAsRead)
	messages.Patch("/:messageId/status", handlers.UpdateMessageStatus)
	messages.Post("/group", handlers.SendGroupMessage)
	messages.Get("/group/:groupId", handlers.GetGroupMessages)

	// Upload routes (protected)
	uploads := api.Group("/upload", middleware.AuthMiddleware)
	uploads.Post("/file", middleware.UploadRateLimiter(), handlers.UploadFile)
	uploads.Post("/avatar", middleware.UploadRateLimiter(), handlers.UploadAvatar)

	// Serve uploaded files (public)
	app.Get("/uploads/:type/:filename", handlers.GetFile)

	// Group routes (protected)
	groups := api.Group("/groups", middleware.AuthMiddleware)
	groups.Post("/", handlers.CreateGroup)
	groups.Get("/", handlers.GetGroups)
	groups.Get("/:groupId", handlers.GetGroupDetails)
	groups.Put("/:groupId", handlers.UpdateGroup)
	groups.Delete("/:groupId", handlers.DeleteGroup)
	groups.Post("/:groupId/members", handlers.AddGroupMembers)
	groups.Delete("/:groupId/members/:userId", handlers.RemoveGroupMember)
	groups.Post("/:groupId/leave", handlers.LeaveGroup)

	// WebSocket route (protected)
	api.Get("/ws", middleware.AuthMiddleware, handlers.WebSocketUpgrade, websocket.New(handlers.WebSocketHandler))

	// WebSocket stats (protected, for debugging)
	api.Get("/ws/stats", middleware.AuthMiddleware, handlers.GetWebSocketStats)
}
