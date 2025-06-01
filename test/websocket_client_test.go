package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

func main() {
	// WebSocket server URL
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/client"}
	log.Printf("Connecting to %s", u.String())

	// Connect to WebSocket
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Channel to handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Channel to handle messages
	done := make(chan struct{})

	// Goroutine to read messages
	go func() {
		defer close(done)
		for {
			var message dto.WebSocketMessage
			err := c.ReadJSON(&message)
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("Received: %s", formatMessage(message))
		}
	}()

	// Test sequence
	go func() {
		// Wait a bit for connection to establish
		time.Sleep(1 * time.Second)

		// Step 1: Authenticate
		log.Println("Step 1: Authenticating...")
		authMsg := dto.WebSocketMessage{
			Type: dto.MessageTypeClientAuth,
			Data: dto.ClientAuthRequest{
				Username: "testclient",
				Password: "password123",
			},
		}

		if err := c.WriteJSON(authMsg); err != nil {
			log.Println("write auth:", err)
			return
		}

		// Wait for auth response
		time.Sleep(2 * time.Second)

		// Step 2: Register PC
		log.Println("Step 2: Registering PC...")
		regMsg := dto.WebSocketMessage{
			Type: dto.MessageTypePCRegistration,
			Data: dto.PCRegistrationRequest{
				PCIdentifier: "test-pc-001",
				IP:           "192.168.1.100",
			},
		}

		if err := c.WriteJSON(regMsg); err != nil {
			log.Println("write registration:", err)
			return
		}

		// Wait for registration response
		time.Sleep(2 * time.Second)

		// Step 3: Send heartbeat
		log.Println("Step 3: Sending heartbeat...")
		hbMsg := dto.WebSocketMessage{
			Type: dto.MessageTypeHeartbeat,
			Data: dto.HeartbeatRequest{
				Timestamp: time.Now().Unix(),
			},
		}

		if err := c.WriteJSON(hbMsg); err != nil {
			log.Println("write heartbeat:", err)
			return
		}

		// Continue sending heartbeats every 30 seconds
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hbMsg := dto.WebSocketMessage{
					Type: dto.MessageTypeHeartbeat,
					Data: dto.HeartbeatRequest{
						Timestamp: time.Now().Unix(),
					},
				}
				if err := c.WriteJSON(hbMsg); err != nil {
					log.Println("write heartbeat:", err)
					return
				}
				log.Println("Heartbeat sent")
			case <-done:
				return
			}
		}
	}()

	// Wait for interrupt signal
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")

			// Cleanly close the connection
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func formatMessage(msg dto.WebSocketMessage) string {
	data, _ := json.MarshalIndent(msg, "", "  ")
	return string(data)
}
