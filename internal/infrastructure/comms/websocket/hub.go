package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub mantiene el conjunto de clientes activos y difunde mensajes a los clientes
type Hub struct {
	// Clientes registrados
	clients map[string]*Client

	// Canal para registrar clientes
	register chan *Client

	// Canal para desregistrar clientes
	unregister chan *Client

	// Canal para mensajes entrantes de clientes
	broadcast chan []byte

	// Mutex para acceso concurrente
	mutex sync.RWMutex

	// Upgrader para conexiones WebSocket
	upgrader websocket.Upgrader
}

// Client representa una conexión WebSocket de un cliente
type Client struct {
	// ID único del cliente (PC ID)
	ID string

	// Conexión WebSocket
	conn *websocket.Conn

	// Canal para mensajes salientes
	send chan []byte

	// Hub al que pertenece
	hub *Hub
}

// NewHub crea una nueva instancia del hub WebSocket
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// En producción, implementar validación de origen apropiada
				return true
			},
		},
	}
}

// Run inicia el hub y maneja los canales
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			h.mutex.Unlock()
			log.Printf("Client %s connected", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.send)
			}
			h.mutex.Unlock()
			log.Printf("Client %s disconnected", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client.ID)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// HandleWebSocket maneja las conexiones WebSocket entrantes
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, clientID string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:   clientID,
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
	}

	client.hub.register <- client

	// Iniciar goroutines para leer y escribir
	go client.writePump()
	go client.readPump()
}

// SendToClient envía un mensaje a un cliente específico
func (h *Hub) SendToClient(ctx context.Context, clientID string, message interface{}) error {
	h.mutex.RLock()
	client, exists := h.clients[clientID]
	h.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not connected", clientID)
	}

	// Serializar mensaje a JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Enviar mensaje al cliente
	select {
	case client.send <- data:
		return nil
	default:
		// Canal lleno, cliente desconectado
		close(client.send)
		delete(h.clients, clientID)
		return fmt.Errorf("client %s send channel full", clientID)
	}
}

// BroadcastToAll envía un mensaje a todos los clientes conectados
func (h *Hub) BroadcastToAll(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.broadcast <- data
	return nil
}

// GetConnectedClients retorna la lista de clientes conectados
func (h *Hub) GetConnectedClients() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clients := make([]string, 0, len(h.clients))
	for clientID := range h.clients {
		clients = append(clients, clientID)
	}
	return clients
}

// IsClientConnected verifica si un cliente está conectado
func (h *Hub) IsClientConnected(clientID string) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, exists := h.clients[clientID]
	return exists
}

// readPump lee mensajes del cliente WebSocket
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Procesar mensaje recibido
		c.handleMessage(message)
	}
}

// writePump escribe mensajes al cliente WebSocket
func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

// handleMessage procesa mensajes recibidos del cliente
func (c *Client) handleMessage(message []byte) {
	// Aquí se procesarían los mensajes entrantes del cliente
	// Por ejemplo: SessionAccepted, SessionRejected, etc.
	log.Printf("Received message from client %s: %s", c.ID, string(message))
} 