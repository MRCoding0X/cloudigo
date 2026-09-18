package handler

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/service"
	"cloudigo/backend/internal/ws"
)

// RegisterWebSocket wires the upload progress/notification socket and the
// admin dashboard's live-activity socket on the same Hub.
func RegisterWebSocket(app *fiber.App, hub *ws.Hub, auth *service.AuthService) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/upload/:uploadId", websocket.New(func(conn *websocket.Conn) {
		room := ws.UploadRoom(conn.Params("uploadId"))
		client := hub.Join(room, conn)
		defer hub.Leave(room, client)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))

	// The browser WebSocket API cannot set an Authorization header, so the
	// access token travels as a query param here instead — short-lived (15
	// min) and admin-only, checked before the upgrade is allowed.
	app.Get("/ws/admin", func(c *fiber.Ctx) error {
		claims, err := auth.ParseAccessToken(c.Query("token"))
		if err != nil || claims.Role != model.RoleAdmin {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		return c.Next()
	}, websocket.New(func(conn *websocket.Conn) {
		client := hub.Join(ws.AdminRoom, conn)
		defer hub.Leave(ws.AdminRoom, client)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
}
