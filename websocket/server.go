// Package websocket provides real-time WebSocket server for the mesh visualization.
// Pushes live path updates and circuit breaker status to frontend clients.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// MsgTypePathUpdate indicates a transaction path update
	MsgTypePathUpdate MessageType = "PATH_UPDATE"
	// MsgTypeCircuitBreaker indicates circuit breaker state change
	MsgTypeCircuitBreaker MessageType = "CIRCUIT_BREAKER"
	// MsgTypeLiquidity indicates liquidity volume change
	MsgTypeLiquidity MessageType = "LIQUIDITY_UPDATE"
	// MsgTypeNodeStatus indicates node status change
	MsgTypeNodeStatus MessageType = "NODE_STATUS"
)

// Message represents a WebSocket message to the frontend
type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// PathUpdate represents a transaction path event
type PathUpdate struct {
	TransactionID string   `json:"transaction_id"`
	Path          []string `json:"path"`
	CurrentHop    int      `json:"current_hop"`
	Amount        int64    `json:"amount"`
	Status        string   `json:"status"` // "in_progress", "completed", "failed", "rerouted"
	OldPath       []string `json:"old_path,omitempty"` // For rerouting visualization
}

// CircuitBreakerEvent represents a circuit breaker state change
type CircuitBreakerEvent struct {
	NodeID    string `json:"node_id"`
	State     string `json:"state"` // "closed", "open", "half_open"
	PrevState string `json:"prev_state,omitempty"`
}

// LiquidityUpdate represents an edge liquidity change
type LiquidityUpdate struct {
	SourceID  string  `json:"source_id"`
	TargetID  string  `json:"target_id"`
	OldVolume int64   `json:"old_volume"`
	NewVolume int64   `json:"new_volume"`
	Change    float64 `json:"change_percent"`
}

// NodeStatusUpdate represents a node status change
type NodeStatusUpdate struct {
	NodeID   string `json:"node_id"`
	IsActive bool   `json:"is_active"`
	Load     int    `json:"load_percent,omitempty"`
}

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client represents a connected WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan *Message
}

// upgrader configures the WebSocket upgrade
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected (total: %d)", len(h.clients))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected (total: %d)", len(h.clients))
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg *Message) {
	msg.Timestamp = time.Now().UnixMilli()
	h.broadcast <- msg
}

// BroadcastPathUpdate sends a path update to all clients
func (h *Hub) BroadcastPathUpdate(update *PathUpdate) {
	h.Broadcast(&Message{
		Type: MsgTypePathUpdate,
		Data: update,
	})
}

// BroadcastCircuitBreaker sends a circuit breaker update
func (h *Hub) BroadcastCircuitBreaker(event *CircuitBreakerEvent) {
	h.Broadcast(&Message{
		Type: MsgTypeCircuitBreaker,
		Data: event,
	})
}

// BroadcastLiquidity sends a liquidity update
func (h *Hub) BroadcastLiquidity(update *LiquidityUpdate) {
	h.Broadcast(&Message{
		Type: MsgTypeLiquidity,
		Data: update,
	})
}

// BroadcastNodeStatus sends a node status update
func (h *Hub) BroadcastNodeStatus(update *NodeStatusUpdate) {
	h.Broadcast(&Message{
		Type: MsgTypeNodeStatus,
		Data: update,
	})
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS handles WebSocket upgrade requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan *Message, 64),
	}

	h.register <- client

	// Start read/write pumps
	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// Server provides the HTTP server for WebSocket connections
type Server struct {
	hub    *Hub
	server *http.Server
}

// NewServer creates a new WebSocket server
func NewServer(addr string) *Server {
	hub := NewHub()
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &Server{
		hub: hub,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Hub returns the WebSocket hub for broadcasting
func (s *Server) Hub() *Hub {
	return s.hub
}

// Start starts the WebSocket server
func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run(ctx)
	log.Printf("WebSocket server starting on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
