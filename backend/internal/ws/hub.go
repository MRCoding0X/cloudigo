// Package ws implements a minimal room-based broadcast hub for server-push
// events over WebSocket (upload/download progress, live download notices).
// Fas 4's admin dashboard live-notifications will reuse the same Hub with an
// "admin" room.
package ws

import (
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*client]struct{})}
}

func (h *Hub) Join(room string, conn *websocket.Conn) *client {
	c := &client{conn: conn}
	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*client]struct{})
	}
	h.rooms[room][c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Leave(room string, c *client) {
	h.mu.Lock()
	delete(h.rooms[room], c)
	if len(h.rooms[room]) == 0 {
		delete(h.rooms, room)
	}
	h.mu.Unlock()
}

// Broadcast sends msg (marshaled as JSON) to every client currently in room.
// Safe to call from any goroutine, including HTTP handlers unrelated to the
// WebSocket connections themselves.
func (h *Hub) Broadcast(room string, msg any) {
	h.mu.RLock()
	clients := make([]*client, 0, len(h.rooms[room]))
	for c := range h.rooms[room] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		err := c.conn.WriteJSON(msg)
		c.mu.Unlock()
		if err != nil {
			log.Printf("ws: broadcast to room %q failed: %v", room, err)
		}
	}
}

func UploadRoom(uploadID string) string {
	return "upload:" + uploadID
}

// AdminRoom is the single shared room admin-dashboard clients join for live
// activity notifications (new upload ready, destroyed, download happened).
const AdminRoom = "admin"
