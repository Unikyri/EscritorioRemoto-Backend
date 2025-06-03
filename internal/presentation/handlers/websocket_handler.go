package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/filetransferservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/videoservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/filetransfer"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/dto"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for now
		// In production, you should validate the origin
		return true
	},
}

// ClientConnection represents an active WebSocket connection
type ClientConnection struct {
	Conn       *websocket.Conn
	UserID     string
	Username   string
	Role       string
	PCID       string
	IsAuth     bool
	LastSeen   time.Time
	RemoteAddr string
}

// WebSocketHandler manages WebSocket connections for client PCs
type WebSocketHandler struct {
	authService         *userservice.AuthService
	pcService           pcservice.IPCService
	sessionService      *remotesessionservice.RemoteSessionService
	videoService        interface{} // VideoService interface
	fileTransferService *filetransferservice.FileTransferService
	adminWSHandler      *AdminWebSocketHandler
	connections         map[string]*ClientConnection // map[connectionID]*ClientConnection
	pcConnections       map[string]*ClientConnection // map[pcID]*ClientConnection
	mutex               sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	authService *userservice.AuthService,
	pcService pcservice.IPCService,
	sessionService *remotesessionservice.RemoteSessionService,
	videoService interface{}, // VideoService interface
	fileTransferService *filetransferservice.FileTransferService,
	adminWSHandler *AdminWebSocketHandler,
) *WebSocketHandler {
	return &WebSocketHandler{
		authService:         authService,
		pcService:           pcService,
		sessionService:      sessionService,
		videoService:        videoService,
		fileTransferService: fileTransferService,
		adminWSHandler:      adminWSHandler,
		connections:         make(map[string]*ClientConnection),
		pcConnections:       make(map[string]*ClientConnection),
		mutex:               sync.RWMutex{},
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Get client IP
	clientIP := getClientIP(c.Request)

	// Create connection object
	connectionID := generateConnectionID()
	clientConn := &ClientConnection{
		Conn:       conn,
		IsAuth:     false,
		LastSeen:   time.Now(),
		RemoteAddr: clientIP,
	}

	// Add to connections map
	h.mutex.Lock()
	h.connections[connectionID] = clientConn
	h.mutex.Unlock()

	// Clean up on exit
	defer func() {
		h.mutex.Lock()
		delete(h.connections, connectionID)
		if clientConn.PCID != "" {
			delete(h.pcConnections, clientConn.PCID)

			// 🔄 Intentar finalizar/rechazar sesiones activas/pendientes para este PC
			log.Printf("⚡ Calling HandleClientPCDisconnect for PCID: %s", clientConn.PCID)
			if err := h.sessionService.HandleClientPCDisconnect(clientConn.PCID); err != nil {
				// Loguear el error, pero no hacer que la desconexión falle por esto.
				// El servicio HandleClientPCDisconnect ya loguea sus propios errores críticos.
				log.Printf("⚠️ Error calling HandleClientPCDisconnect for PC %s: %v", clientConn.PCID, err)
			}

			// Marcar PC como offline cuando se cierra la conexión
			if clientConn.IsAuth {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Obtener información del PC antes de marcarlo como offline
				currentPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
				var pcIdentifier string
				if err == nil && currentPC != nil {
					pcIdentifier = currentPC.Identifier
				}

				// Obtener estado anterior del PC para notificación
				oldStatus := "ONLINE"

				err = h.pcService.UpdatePCConnectionStatus(ctx, clientConn.PCID, clientpc.PCConnectionStatusOffline)
				if err != nil {
					log.Printf("Error updating PC status to offline: %v", err)
				} else {
					log.Printf("PC marked as offline: %s (%s)", clientConn.PCID, clientConn.Username)

					// Notificar a administradores sobre la desconexión del PC
					if h.adminWSHandler != nil {
						h.adminWSHandler.BroadcastPCDisconnected(clientConn.PCID, pcIdentifier, clientConn.UserID)
						// Notificar cambio de estado específico
						h.adminWSHandler.BroadcastPCStatusChanged(clientConn.PCID, pcIdentifier, oldStatus, "OFFLINE")
						// Notificar actualización general de la lista
						h.adminWSHandler.BroadcastPCListUpdate()
					}
				}
			}
		}
		h.mutex.Unlock()
		log.Printf("Client disconnected: %s (Username: %s, PCID: %s)", connectionID, clientConn.Username, clientConn.PCID)
	}()

	log.Printf("New WebSocket connection: %s from %s", connectionID, clientIP)

	// Handle messages
	for {
		var message dto.WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Update last seen
		clientConn.LastSeen = time.Now()

		// Handle message based on type
		switch message.Type {
		case dto.MessageTypeClientAuth:
			h.handleClientAuth(conn, clientConn, message.Data)
		case dto.MessageTypePCRegistration:
			h.handlePCRegistration(conn, clientConn, message.Data, clientIP)
		case dto.MessageTypeHeartbeat:
			h.handleHeartbeat(conn, clientConn, message.Data)
		case dto.MessageTypeScreenFrame:
			h.handleScreenFrame(conn, clientConn, message.Data)
		case "session_accepted":
			h.handleSessionAccepted(conn, clientConn, message.Data)
		case "session_rejected":
			h.handleSessionRejected(conn, clientConn, message.Data)
		case "video_chunk_upload":
			h.handleVideoChunkUpload(conn, clientConn, message.Data)
		case "video_upload_complete":
			h.handleVideoUploadComplete(conn, clientConn, message.Data)
		case "video_frame_upload":
			h.handleVideoFrameUpload(conn, clientConn, message.Data)
		case "video_recording_complete":
			h.handleVideoRecordingComplete(conn, clientConn, message.Data)
		case "file_transfer_ack":
			h.handleFileTransferAcknowledgement(conn, clientConn, message.Data)
		default:
			log.Printf("Unknown message type: %s", message.Type)
		}
	}
}

// handleClientAuth handles client authentication
func (h *WebSocketHandler) handleClientAuth(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse authentication request
	authData, err := json.Marshal(data)
	if err != nil {
		h.sendAuthResponse(conn, false, "", "", "Invalid request format")
		return
	}

	var authReq dto.ClientAuthRequest
	if err := json.Unmarshal(authData, &authReq); err != nil {
		h.sendAuthResponse(conn, false, "", "", "Invalid request format")
		return
	}

	// Authenticate user
	token, user, err := h.authService.AuthenticateClient(authReq.Username, authReq.Password)
	if err != nil {
		h.sendAuthResponse(conn, false, "", "", "Authentication failed")
		return
	}

	// Update connection with user info
	clientConn.UserID = user.UserID()
	clientConn.Username = user.Username()
	clientConn.Role = string(user.Role())
	clientConn.IsAuth = true

	// Send success response
	h.sendAuthResponse(conn, true, token, user.UserID(), "")
	log.Printf("Client authenticated: %s (%s)", user.Username(), user.UserID())
}

// handlePCRegistration handles PC registration
func (h *WebSocketHandler) handlePCRegistration(conn *websocket.Conn, clientConn *ClientConnection, data interface{}, clientIP string) {
	// Check if client is authenticated
	if !clientConn.IsAuth {
		h.sendPCRegistrationResponse(conn, false, "", "Authentication required")
		return
	}

	// Parse registration request
	regData, err := json.Marshal(data)
	if err != nil {
		h.sendPCRegistrationResponse(conn, false, "", "Invalid request format")
		return
	}

	var regReq dto.PCRegistrationRequest
	if err := json.Unmarshal(regData, &regReq); err != nil {
		h.sendPCRegistrationResponse(conn, false, "", "Invalid request format")
		return
	}

	// Use provided IP or detect from connection
	ip := regReq.IP
	if ip == "" {
		ip = clientIP
	}

	// Register PC
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pc, err := h.pcService.RegisterPC(ctx, clientConn.UserID, regReq.PCIdentifier, ip)
	if err != nil {
		h.sendPCRegistrationResponse(conn, false, "", err.Error())
		return
	}

	// Update connection with PC info
	clientConn.PCID = pc.PCID

	// Add to PC connections map
	h.mutex.Lock()
	h.pcConnections[pc.PCID] = clientConn
	h.mutex.Unlock()

	// Notificar a administradores sobre el registro del PC
	if h.adminWSHandler != nil {
		h.adminWSHandler.BroadcastPCRegistered(pc.PCID, pc.Identifier, pc.OwnerUserID, pc.IP)

		// También notificar que el PC está ahora ONLINE ya que se acaba de conectar
		log.Printf("PC registered and online: %s (%s) for user %s", pc.Identifier, pc.PCID, clientConn.Username)
		h.adminWSHandler.BroadcastPCConnected(pc.PCID, pc.Identifier, pc.OwnerUserID, pc.IP)
		h.adminWSHandler.BroadcastPCStatusChanged(pc.PCID, pc.Identifier, "OFFLINE", "ONLINE")

		// Notificar actualización general de la lista
		h.adminWSHandler.BroadcastPCListUpdate()
	}

	// Send success response
	h.sendPCRegistrationResponse(conn, true, pc.PCID, "")
	log.Printf("PC registered: %s (%s) for user %s", regReq.PCIdentifier, pc.PCID, clientConn.Username)
}

// handleHeartbeat handles heartbeat messages
func (h *WebSocketHandler) handleHeartbeat(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse heartbeat request
	hbData, err := json.Marshal(data)
	if err != nil {
		return
	}

	var hbReq dto.HeartbeatRequest
	if err := json.Unmarshal(hbData, &hbReq); err != nil {
		return
	}

	// Update last seen and ensure PC is online if PC is registered
	if clientConn.PCID != "" && clientConn.IsAuth {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Primero verificar el estado actual del PC para detectar cambios
		currentPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
		var wasOffline bool
		if err == nil && currentPC != nil {
			wasOffline = currentPC.ConnectionStatus == clientpc.PCConnectionStatusOffline
		}

		// Actualizar último visto
		if err := h.pcService.UpdatePCLastSeen(ctx, clientConn.PCID); err != nil {
			log.Printf("Error updating PC last seen: %v", err)
		}

		// Asegurar que el PC esté marcado como online
		if err := h.pcService.UpdatePCConnectionStatus(ctx, clientConn.PCID, clientpc.PCConnectionStatusOnline); err != nil {
			log.Printf("Error updating PC status to online: %v", err)
		} else {
			// Si el PC estaba offline y ahora está online, notificar el cambio
			if wasOffline && h.adminWSHandler != nil {
				log.Printf("PC reconnected: %s (%s)", clientConn.PCID, clientConn.Username)

				// Obtener información actualizada del PC para las notificaciones
				updatedPC, err := h.pcService.GetPCByID(ctx, clientConn.PCID)
				if err == nil && updatedPC != nil {
					// Notificar reconexión del PC con información completa
					h.adminWSHandler.BroadcastPCConnected(updatedPC.PCID, updatedPC.Identifier, updatedPC.OwnerUserID, updatedPC.IP)
					// Notificar cambio de estado específico
					h.adminWSHandler.BroadcastPCStatusChanged(updatedPC.PCID, updatedPC.Identifier, "OFFLINE", "ONLINE")
					// Notificar actualización general de la lista
					h.adminWSHandler.BroadcastPCListUpdate()
				}
			}

			// Procesar transferencias pendientes para este cliente
			log.Printf("🔍 Verificando transferencias pendientes para cliente: %s", clientConn.PCID)
			h.processPendingTransfers(clientConn.PCID)
		}
	}

	// Send heartbeat response
	response := dto.WebSocketMessage{
		Type: dto.MessageTypeHeartbeatResp,
		Data: dto.HeartbeatResponse{
			Timestamp: time.Now().Unix(),
			Status:    "OK",
		},
	}

	conn.WriteJSON(response)
}

// handleScreenFrame maneja frames de pantalla recibidos de clientes
func (h *WebSocketHandler) handleScreenFrame(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	if !clientConn.IsAuth {
		log.Printf("❌ SCREEN FRAME: Unauthorized client attempted to send frame")
		return
	}

	if clientConn.PCID == "" {
		log.Printf("❌ SCREEN FRAME: Client not registered, cannot process frame")
		return
	}

	// Parse screen frame data
	frameData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Error marshalling frame data: %v", err)
		return
	}

	var screenFrame dto.ScreenFrame
	if err := json.Unmarshal(frameData, &screenFrame); err != nil {
		log.Printf("❌ SCREEN FRAME: Error unmarshalling screen frame: %v", err)
		return
	}

	log.Printf("📹 SCREEN FRAME: Received frame %d from PC %s (session: %s, size: %dx%d)",
		screenFrame.SequenceNum, clientConn.PCID, screenFrame.SessionID, screenFrame.Width, screenFrame.Height)

	// Validar que la sesión está activa y el PC tiene permisos
	err = h.sessionService.ValidateStreamingPermission(screenFrame.SessionID, clientConn.PCID)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Invalid streaming permission: %v", err)
		return
	}

	// Obtener el administrador que está controlando esta sesión
	adminUserID, err := h.sessionService.GetAdminUserIDForActiveSession(screenFrame.SessionID)
	if err != nil {
		log.Printf("❌ SCREEN FRAME: Error getting admin for session: %v", err)
		return
	}

	// Reenviar frame al administrador a través del AdminWebSocketHandler
	if h.adminWSHandler != nil {
		err := h.adminWSHandler.ForwardScreenFrameToAdmin(adminUserID, screenFrame)
		if err != nil {
			log.Printf("❌ SCREEN FRAME: Error forwarding frame to admin %s: %v", adminUserID, err)
		} else {
			log.Printf("✅ SCREEN FRAME: Frame %d forwarded to admin %s", screenFrame.SequenceNum, adminUserID)
		}
	} else {
		log.Printf("⚠️ SCREEN FRAME: No admin WebSocket handler available")
	}
}

// handleSessionAccepted maneja cuando el cliente acepta una sesión de control remoto
func (h *WebSocketHandler) handleSessionAccepted(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse session accepted message
	sessionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling session accepted data: %v", err)
		return
	}

	var acceptedMsg struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(sessionData, &acceptedMsg); err != nil {
		log.Printf("Error unmarshalling session accepted message: %v", err)
		return
	}

	log.Printf("🎉 Client %s accepted remote control session: %s", clientConn.PCID, acceptedMsg.SessionID)

	// Actualizar estado de sesión en base de datos a ACTIVE
	err = h.sessionService.AcceptSession(acceptedMsg.SessionID)
	if err != nil {
		log.Printf("❌ Error accepting session in service: %v", err)

		// Enviar error al cliente
		errorMsg := dto.WebSocketMessage{
			Type: "session_failed",
			Data: map[string]interface{}{
				"session_id": acceptedMsg.SessionID,
				"error":      "Failed to activate session",
				"message":    err.Error(),
			},
		}
		conn.WriteJSON(errorMsg)
		return
	}

	log.Printf("✅ Session %s successfully activated in database", acceptedMsg.SessionID)

	// Notificar al administrador que la sesión fue aceptada
	if h.adminWSHandler != nil {
		err = h.adminWSHandler.NotifySessionAccepted(acceptedMsg.SessionID)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to notify admin of session acceptance: %v", err)
		} else {
			log.Printf("✅ Admin notified of session %s acceptance", acceptedMsg.SessionID)
		}
	}

	// Enviar confirmación de sesión iniciada al cliente
	sessionStartedMsg := dto.WebSocketMessage{
		Type: "session_started",
		Data: map[string]interface{}{
			"session_id": acceptedMsg.SessionID,
			"status":     "ACTIVE",
			"message":    "Remote control session started successfully",
			"timestamp":  time.Now().Unix(),
		},
	}

	err = conn.WriteJSON(sessionStartedMsg)
	if err != nil {
		log.Printf("Error sending session started message to client: %v", err)
	} else {
		log.Printf("✅ Session started confirmation sent to client %s", clientConn.PCID)
	}
}

// handleSessionRejected maneja cuando el cliente rechaza una sesión de control remoto
func (h *WebSocketHandler) handleSessionRejected(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse session rejected message
	sessionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling session rejected data: %v", err)
		return
	}

	var rejectedMsg struct {
		SessionID string `json:"session_id"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(sessionData, &rejectedMsg); err != nil {
		log.Printf("Error unmarshalling session rejected message: %v", err)
		return
	}

	log.Printf("❌ Client %s rejected remote control session: %s (reason: %s)",
		clientConn.PCID, rejectedMsg.SessionID, rejectedMsg.Reason)

	// TODO: Actualizar estado de sesión en base de datos a ENDED_BY_CLIENT
	// TODO: Notificar al administrador que la sesión fue rechazada
}

// handleVideoChunkUpload maneja la subida de chunks de video
func (h *WebSocketHandler) handleVideoChunkUpload(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Verificar autenticación
	if !clientConn.IsAuth {
		log.Printf("❌ VIDEO CHUNK UPLOAD: Unauthorized client attempted video upload")
		return
	}

	// Verificar que tenemos videoService
	if h.videoService == nil {
		log.Printf("❌ VIDEO CHUNK UPLOAD: VideoService not available")
		response := dto.WebSocketMessage{
			Type: "video_upload_error",
			Data: map[string]interface{}{
				"error": "Video service not available",
			},
		}
		conn.WriteJSON(response)
		return
	}

	// Parsear datos del chunk de video
	chunkData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ VIDEO CHUNK UPLOAD: Error marshalling chunk data: %v", err)
		return
	}

	var videoChunk dto.VideoChunk
	if err := json.Unmarshal(chunkData, &videoChunk); err != nil {
		log.Printf("❌ VIDEO CHUNK UPLOAD: Error unmarshalling video chunk: %v", err)
		return
	}

	log.Printf("📹 VIDEO CHUNK UPLOAD: Received video chunk %d from PC %s (video: %s)",
		videoChunk.ChunkIndex, clientConn.PCID, videoChunk.VideoID)

	// 🚀 PROCESAR CHUNK REAL USANDO VIDEOSERVICE
	if h.videoService != nil {
		// Decodificar chunk data de base64 a bytes
		chunkData, err := base64.StdEncoding.DecodeString(videoChunk.ChunkData)
		if err != nil {
			log.Printf("❌ VIDEO CHUNK UPLOAD: Error decoding chunk data: %v", err)

			errorResponse := dto.WebSocketMessage{
				Type: "video_upload_error",
				Data: map[string]interface{}{
					"video_id": videoChunk.VideoID,
					"error":    "Error decoding chunk data",
				},
			}
			conn.WriteJSON(errorResponse)
			return
		}

		// Convertir a formato del VideoService
		serviceChunk := videoservice.VideoChunk{
			VideoID:     videoChunk.VideoID,
			SessionID:   videoChunk.SessionID,
			ChunkIndex:  videoChunk.ChunkIndex,
			ChunkData:   chunkData, // Ahora es []byte
			IsLastChunk: videoChunk.IsLastChunk,
			FileSize:    videoChunk.FileSize,
			Duration:    videoChunk.Duration,
			FileName:    videoChunk.FileName,
		}

		// Procesar chunk usando VideoService (sin type cast necesario)
		result, err := h.videoService.(videoservice.IVideoService).HandleUploadedVideoChunk(serviceChunk)
		if err != nil {
			log.Printf("❌ VIDEO CHUNK UPLOAD: Error processing chunk: %v", err)

			// Enviar respuesta de error
			errorResponse := dto.WebSocketMessage{
				Type: "video_upload_error",
				Data: map[string]interface{}{
					"video_id": videoChunk.VideoID,
					"error":    err.Error(),
				},
			}
			conn.WriteJSON(errorResponse)
			return
		}

		log.Printf("✅ VIDEO CHUNK UPLOAD: Chunk %d processed successfully for video %s",
			videoChunk.ChunkIndex, videoChunk.VideoID)

		// Enviar respuesta apropiada según el resultado
		if result != nil && result.IsComplete {
			// Video completo - enviar respuesta final
			log.Printf("🎉 VIDEO COMPLETE: Video %s upload completed and saved to %s",
				videoChunk.VideoID, result.FilePath)

			successResponse := dto.WebSocketMessage{
				Type: "video_upload_completed",
				Data: dto.VideoUploadComplete{
					VideoID:   videoChunk.VideoID,
					SessionID: videoChunk.SessionID,
					FilePath:  result.FilePath,
					Duration:  result.Duration,
					FileSize:  result.FileSize,
					Message:   "Video uploaded and processed successfully",
				},
			}

			if err := conn.WriteJSON(successResponse); err != nil {
				log.Printf("❌ VIDEO CHUNK UPLOAD: Error sending success response: %v", err)
			}
		} else {
			// Progreso parcial - enviar actualización
			progressResponse := dto.WebSocketMessage{
				Type: "video_upload_progress",
				Data: dto.VideoUploadProgress{
					VideoID:         videoChunk.VideoID,
					ChunksReceived:  result.ChunksReceived,
					TotalChunks:     result.TotalChunks,
					ProgressPercent: result.ProgressPercent,
				},
			}

			if err := conn.WriteJSON(progressResponse); err != nil {
				log.Printf("❌ VIDEO CHUNK UPLOAD: Error sending progress response: %v", err)
			}
		}
	} else {
		log.Printf("⚠️ VIDEO CHUNK UPLOAD: VideoService not available, using fallback response")

		// Fallback response si VideoService no está disponible
		if videoChunk.IsLastChunk {
			successResponse := dto.WebSocketMessage{
				Type: "video_upload_completed",
				Data: dto.VideoUploadComplete{
					VideoID:   videoChunk.VideoID,
					SessionID: videoChunk.SessionID,
					FilePath:  fmt.Sprintf("videos/processed/%s_%s.mp4", videoChunk.SessionID, videoChunk.VideoID),
					Duration:  videoChunk.Duration,
					FileSize:  videoChunk.FileSize,
					Message:   "Video upload acknowledged (VideoService unavailable)",
				},
			}
			conn.WriteJSON(successResponse)
		} else {
			progressResponse := dto.WebSocketMessage{
				Type: "video_upload_progress",
				Data: dto.VideoUploadProgress{
					VideoID:         videoChunk.VideoID,
					ChunksReceived:  videoChunk.ChunkIndex + 1,
					TotalChunks:     10, // Valor por defecto
					ProgressPercent: float64(videoChunk.ChunkIndex+1) * 10,
				},
			}
			conn.WriteJSON(progressResponse)
		}
	}
}

// handleVideoUploadComplete handles video upload completion
func (h *WebSocketHandler) handleVideoUploadComplete(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse video upload completion message
	completionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling video upload completion data: %v", err)
		return
	}

	var completionMsg struct {
		VideoID string `json:"video_id"`
	}
	if err := json.Unmarshal(completionData, &completionMsg); err != nil {
		log.Printf("Error unmarshalling video upload completion message: %v", err)
		return
	}

	log.Printf("🎉 Video %s upload completed", completionMsg.VideoID)

	// Send confirmation to client
	completionConfirmedMsg := dto.WebSocketMessage{
		Type: "video_upload_completed_confirmed",
		Data: map[string]interface{}{
			"video_id":  completionMsg.VideoID,
			"message":   "Video upload completed and confirmed",
			"timestamp": time.Now().Unix(),
		},
	}

	err = conn.WriteJSON(completionConfirmedMsg)
	if err != nil {
		log.Printf("Error sending video upload completion confirmation to client: %v", err)
	} else {
		log.Printf("✅ Video upload completion confirmation sent to client %s", clientConn.PCID)
	}
}

// handleVideoFrameUpload handles individual JPEG frame uploads for the new frame-based recording system
func (h *WebSocketHandler) handleVideoFrameUpload(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Verificar autenticación
	if !clientConn.IsAuth {
		log.Printf("❌ VIDEO FRAME UPLOAD: Unauthorized client attempted frame upload")
		return
	}

	// Parsear datos del frame de video
	frameData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ VIDEO FRAME UPLOAD: Error marshalling frame data: %v", err)
		return
	}

	var videoFrame struct {
		SessionID  string `json:"session_id"`
		VideoID    string `json:"video_id"`
		FrameIndex int    `json:"frame_index"`
		Timestamp  int64  `json:"timestamp"`
		FrameData  string `json:"frame_data"` // Base64 encoded JPEG
	}

	if err := json.Unmarshal(frameData, &videoFrame); err != nil {
		log.Printf("❌ VIDEO FRAME UPLOAD: Error unmarshalling frame data: %v", err)
		return
	}

	log.Printf("📸 VIDEO FRAME UPLOAD: Received frame %d from PC %s (session: %s, video: %s)",
		videoFrame.FrameIndex, clientConn.PCID, videoFrame.SessionID, videoFrame.VideoID)

	// Verificar que tenemos videoService
	if h.videoService == nil {
		log.Printf("❌ VIDEO FRAME UPLOAD: VideoService not available")
		return
	}

	// Decodificar frame data de base64 a bytes
	frameBytes, err := base64.StdEncoding.DecodeString(videoFrame.FrameData)
	if err != nil {
		log.Printf("❌ VIDEO FRAME UPLOAD: Error decoding frame data: %v", err)
		return
	}

	// Procesar frame usando VideoService
	frameInfo := videoservice.VideoFrameInfo{
		VideoID:    videoFrame.VideoID,
		SessionID:  videoFrame.SessionID,
		FrameIndex: videoFrame.FrameIndex,
		Timestamp:  videoFrame.Timestamp,
		FrameData:  frameBytes,
	}

	err = h.videoService.(videoservice.IVideoService).SaveVideoFrame(frameInfo)
	if err != nil {
		log.Printf("❌ VIDEO FRAME UPLOAD: Error saving frame %d: %v", videoFrame.FrameIndex, err)
		return
	}

	// Solo loguear cada 30 frames para no saturar
	if videoFrame.FrameIndex%30 == 0 || videoFrame.FrameIndex == 0 {
		log.Printf("✅ VIDEO FRAME UPLOAD: Frame %d saved successfully for video %s",
			videoFrame.FrameIndex, videoFrame.VideoID)
	}
}

// handleVideoRecordingComplete handles the completion metadata for frame-based recordings
func (h *WebSocketHandler) handleVideoRecordingComplete(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Verificar autenticación
	if !clientConn.IsAuth {
		log.Printf("❌ VIDEO RECORDING COMPLETE: Unauthorized client attempted to complete recording")
		return
	}

	// Parsear datos de finalización
	completionData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ VIDEO RECORDING COMPLETE: Error marshalling completion data: %v", err)
		return
	}

	var recordingComplete struct {
		VideoID         string  `json:"video_id"`
		SessionID       string  `json:"session_id"`
		TotalFrames     int     `json:"total_frames"`
		FPS             float64 `json:"fps"`
		DurationSeconds float64 `json:"duration_seconds"`
		Timestamp       int64   `json:"timestamp"`
	}

	if err := json.Unmarshal(completionData, &recordingComplete); err != nil {
		log.Printf("❌ VIDEO RECORDING COMPLETE: Error unmarshalling completion data: %v", err)
		return
	}

	log.Printf("🎬 VIDEO RECORDING COMPLETE: Recording finished - VideoID: %s, SessionID: %s, Frames: %d, FPS: %.2f, Duration: %.2fs",
		recordingComplete.VideoID, recordingComplete.SessionID, recordingComplete.TotalFrames, recordingComplete.FPS, recordingComplete.DurationSeconds)

	// Verificar que tenemos videoService
	if h.videoService == nil {
		log.Printf("❌ VIDEO RECORDING COMPLETE: VideoService not available")
		return
	}

	// Procesar finalización usando VideoService
	recordingInfo := videoservice.VideoRecordingMetadata{
		VideoID:         recordingComplete.VideoID,
		SessionID:       recordingComplete.SessionID,
		TotalFrames:     recordingComplete.TotalFrames,
		FPS:             recordingComplete.FPS,
		DurationSeconds: recordingComplete.DurationSeconds,
		CompletedAt:     time.Now(),
	}

	err = h.videoService.(videoservice.IVideoService).FinalizeVideoRecording(recordingInfo)
	if err != nil {
		log.Printf("❌ VIDEO RECORDING COMPLETE: Error finalizing recording: %v", err)
		return
	}

	log.Printf("✅ VIDEO RECORDING COMPLETE: Recording %s finalized successfully with %d frames",
		recordingComplete.VideoID, recordingComplete.TotalFrames)

	// Enviar confirmación al cliente
	confirmationMsg := dto.WebSocketMessage{
		Type: "video_recording_finalized",
		Data: map[string]interface{}{
			"video_id":     recordingComplete.VideoID,
			"session_id":   recordingComplete.SessionID,
			"total_frames": recordingComplete.TotalFrames,
			"duration":     recordingComplete.DurationSeconds,
			"message":      "Recording finalized successfully",
			"timestamp":    time.Now().Unix(),
		},
	}

	err = conn.WriteJSON(confirmationMsg)
	if err != nil {
		log.Printf("❌ VIDEO RECORDING COMPLETE: Error sending confirmation to client: %v", err)
	} else {
		log.Printf("📤 VIDEO RECORDING COMPLETE: Confirmation sent to client %s", clientConn.PCID)
	}
}

// handleFileTransferAcknowledgement handles file transfer acknowledgement messages
func (h *WebSocketHandler) handleFileTransferAcknowledgement(conn *websocket.Conn, clientConn *ClientConnection, data interface{}) {
	// Parse file transfer acknowledgement message
	ackData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling file transfer acknowledgement data: %v", err)
		return
	}

	var ackMsg dto.FileTransferAcknowledgement
	if err := json.Unmarshal(ackData, &ackMsg); err != nil {
		log.Printf("Error unmarshalling file transfer acknowledgement message: %v", err)
		return
	}

	log.Printf("✅ FILE TRANSFER ACK: Received acknowledgement for transfer %s with status: %s",
		ackMsg.TransferID, ackMsg.Status)

	// Actualizar estado en base de datos según el tipo de acknowledgement
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch ackMsg.Status {
	case "READY":
		// Cliente está listo para recibir el archivo
		log.Printf("📡 Client ready for transfer %s", ackMsg.TransferID)

	case "CHUNK_RECEIVED":
		// Cliente confirmó recepción de un chunk
		log.Printf("📦 Chunk %d received by client for transfer %s", ackMsg.ChunkNumber, ackMsg.TransferID)

	case "COMPLETED_CLIENT":
		// Cliente confirmó que recibió todo el archivo exitosamente
		err := h.fileTransferService.UpdateTransferStatus(
			ctx,
			ackMsg.TransferID,
			filetransfer.TransferStatusCompleted,
			"",
		)
		if err != nil {
			log.Printf("Error updating transfer status to COMPLETED: %v", err)
		} else {
			log.Printf("🎉 Transfer %s completed successfully", ackMsg.TransferID)

			// Notificar al administrador
			if h.adminWSHandler != nil {
				// Obtener detalles de la transferencia para la notificación
				_, err := h.fileTransferService.GetTransferByID(ctx, ackMsg.TransferID)
				if err == nil {
					// TODO: Implementar BroadcastFileTransferCompleted en AdminWebSocketHandler
					// h.adminWSHandler.BroadcastFileTransferCompleted(
					// 	transfer.TransferID(),
					// 	transfer.FileName(),
					// 	transfer.TargetPCID(),
					// )
				}
			}
		}

	case "FAILED_CLIENT":
		// Cliente reportó error en la transferencia
		errorMsg := ackMsg.ErrorMessage
		if errorMsg == "" {
			errorMsg = "Error reportado por el cliente"
		}

		err := h.fileTransferService.UpdateTransferStatus(
			ctx,
			ackMsg.TransferID,
			filetransfer.TransferStatusFailed,
			errorMsg,
		)
		if err != nil {
			log.Printf("Error updating transfer status to FAILED: %v", err)
		} else {
			log.Printf("❌ Transfer %s failed: %s", ackMsg.TransferID, errorMsg)

			// Notificar al administrador
			if h.adminWSHandler != nil {
				// TODO: Implementar BroadcastFileTransferFailed en AdminWebSocketHandler
				// h.adminWSHandler.BroadcastFileTransferFailed(ackMsg.TransferID, errorMsg)
			}
		}

	default:
		log.Printf("⚠️ Unknown acknowledgement status: %s for transfer %s", ackMsg.Status, ackMsg.TransferID)
	}
}

// Helper methods for sending responses

func (h *WebSocketHandler) sendAuthResponse(conn *websocket.Conn, success bool, token, userID, errorMsg string) {
	response := dto.WebSocketMessage{
		Type: dto.MessageTypeClientAuthResp,
		Data: dto.ClientAuthResponse{
			Success: success,
			Token:   token,
			UserID:  userID,
			Error:   errorMsg,
		},
	}
	conn.WriteJSON(response)
}

func (h *WebSocketHandler) sendPCRegistrationResponse(conn *websocket.Conn, success bool, pcID, errorMsg string) {
	response := dto.WebSocketMessage{
		Type: dto.MessageTypePCRegistrationResp,
		Data: dto.PCRegistrationResponse{
			Success: success,
			PCID:    pcID,
			Error:   errorMsg,
		},
	}
	conn.WriteJSON(response)
}

// GetConnectedPCs returns a list of currently connected PCs
func (h *WebSocketHandler) GetConnectedPCs() map[string]*ClientConnection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make(map[string]*ClientConnection)
	for pcID, conn := range h.pcConnections {
		result[pcID] = conn
	}
	return result
}

// Utility functions

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func generateConnectionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// SendRemoteControlRequestToClient envía una solicitud de control remoto a un cliente específico
func (h *WebSocketHandler) SendRemoteControlRequestToClient(sessionID, clientPCID, adminUserID, adminUsername string) error {
	log.Printf("🚀 REMOTE CONTROL: Attempting to send request to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("❌ REMOTE CONTROL: Client PC %s not found in connections map", clientPCID)
		log.Printf("📊 REMOTE CONTROL: Current connected PCs: %d", len(h.pcConnections))
		for pcID := range h.pcConnections {
			log.Printf("  - Connected PC: %s", pcID)
		}
		return fmt.Errorf("client PC %s not connected", clientPCID)
	}

	log.Printf("✅ REMOTE CONTROL: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de solicitud de control remoto
	remoteControlMsg := dto.WebSocketMessage{
		Type: "remote_control_request",
		Data: map[string]interface{}{
			"session_id":     sessionID,
			"admin_user_id":  adminUserID,
			"admin_username": adminUsername,
			"client_pc_id":   clientPCID,
			"timestamp":      time.Now().Unix(),
		},
	}

	log.Printf("📡 REMOTE CONTROL: Sending message to client %s: %+v", clientPCID, remoteControlMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(remoteControlMsg)
	if err != nil {
		log.Printf("❌ REMOTE CONTROL: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ REMOTE CONTROL: Request sent successfully to client %s (session: %s)", clientPCID, sessionID)
	return nil
}

// SendInputCommandToClient envía un comando de input a un cliente específico
func (h *WebSocketHandler) SendInputCommandToClient(clientPCID string, inputCommand dto.InputCommand) error {
	log.Printf("🖱️ REMOTE CONTROL: Attempting to send input command to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("❌ INPUT COMMAND: Client PC %s not found in connections map", clientPCID)
		return fmt.Errorf("client PC %s not connected", clientPCID)
	}

	log.Printf("✅ INPUT COMMAND: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de comando de input
	inputMsg := dto.WebSocketMessage{
		Type: "input_command",
		Data: inputCommand,
	}

	log.Printf("📡 INPUT COMMAND: Sending to client %s: %+v", clientPCID, inputMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(inputMsg)
	if err != nil {
		log.Printf("❌ INPUT COMMAND: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ INPUT COMMAND: Sent successfully to client %s", clientPCID)
	return nil
}

// === FILE TRANSFER METHODS ===

// SendFileTransferRequestToClient sends a file transfer request to the specified client PC
func (h *WebSocketHandler) SendFileTransferRequestToClient(transfer *filetransfer.FileTransfer) error {
	h.mutex.RLock()
	clientConn, exists := h.pcConnections[transfer.TargetPCID()]
	h.mutex.RUnlock()

	if !exists {
		// Cliente no está conectado, marcar transferencia como fallida
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.fileTransferService.UpdateTransferStatus(
			ctx,
			transfer.TransferID(),
			filetransfer.TransferStatusFailed,
			"Cliente no está conectado",
		)
		if err != nil {
			log.Printf("Error updating transfer status for offline client: %v", err)
		}

		return fmt.Errorf("client PC not connected: %s", transfer.TargetPCID())
	}

	// Calcular total de chunks para el archivo
	chunkSize := 64 * 1024                                                   // 64KB chunks
	fileSize := int64(transfer.FileSizeMB() * 1024 * 1024)                   // Convertir MB a bytes
	totalChunks := int((fileSize + int64(chunkSize) - 1) / int64(chunkSize)) // Redondear hacia arriba

	// Crear mensaje de solicitud de transferencia con estructura actualizada
	request := dto.FileTransferRequest{
		Type:            "file_transfer_request",
		TransferID:      transfer.TransferID(),
		SessionID:       transfer.AssociatedSessionID(),
		FileName:        transfer.FileName(),
		FileSize:        fileSize,
		FileSizeMB:      transfer.FileSizeMB(),
		TotalChunks:     totalChunks,
		DestinationPath: transfer.DestinationPathClient(),
		InitiatedBy:     transfer.InitiatingUserID(),
		Timestamp:       time.Now().Unix(), // Unix timestamp
	}

	message := dto.WebSocketMessage{
		Type: "file_transfer_request",
		Data: request,
	}

	if err := clientConn.Conn.WriteJSON(message); err != nil {
		log.Printf("Error sending file transfer request to client PC %s: %v", transfer.TargetPCID(), err)

		// Marcar transferencia como fallida
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		h.fileTransferService.UpdateTransferStatus(
			ctx,
			transfer.TransferID(),
			filetransfer.TransferStatusFailed,
			fmt.Sprintf("Error enviando solicitud: %v", err),
		)

		return err
	}

	log.Printf("✅ File transfer request sent to client PC: %s (Transfer: %s, File: %s, %d chunks)",
		transfer.TargetPCID(), transfer.TransferID(), transfer.FileName(), totalChunks)
	return nil
}

// SendFileChunksToClient sends file chunks to the client PC
func (h *WebSocketHandler) SendFileChunksToClient(transfer *filetransfer.FileTransfer) error {
	// Actualizar estado a IN_PROGRESS
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.fileTransferService.UpdateTransferStatus(
		ctx,
		transfer.TransferID(),
		filetransfer.TransferStatusInProgress,
		"",
	)
	if err != nil {
		log.Printf("Error updating transfer status to IN_PROGRESS: %v", err)
	}

	// Verificar que el cliente sigue conectado
	h.mutex.RLock()
	clientConn, exists := h.pcConnections[transfer.TargetPCID()]
	h.mutex.RUnlock()

	if !exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.fileTransferService.UpdateTransferStatus(
			ctx,
			transfer.TransferID(),
			filetransfer.TransferStatusFailed,
			"Cliente se desconectó durante la transferencia",
		)
		if err != nil {
			log.Printf("Error updating transfer status for disconnected client: %v", err)
		}

		return fmt.Errorf("client PC disconnected during transfer: %s", transfer.TargetPCID())
	}

	// Calcular total de chunks
	chunkSize := 64 * 1024 // 64KB chunks
	fileSize := int64(transfer.FileSizeMB() * 1024 * 1024)
	totalChunks := int((fileSize + int64(chunkSize) - 1) / int64(chunkSize))
	chunkIndex := 0

	// Leer archivo en chunks y enviar
	err = h.fileTransferService.ReadFileInChunks(
		transfer.SourcePathServer(),
		chunkSize,
		func(chunkData []byte, isLastChunk bool) error {
			// Codificar chunk en base64
			encodedData := base64.StdEncoding.EncodeToString(chunkData)

			// Usar estructura actualizada
			chunk := dto.FileChunk{
				Type:          "file_chunk",
				TransferID:    transfer.TransferID(),
				SessionID:     transfer.AssociatedSessionID(),
				ChunkIndex:    chunkIndex, // 0-based index
				TotalChunks:   totalChunks,
				ChunkData:     encodedData, // Base64 string directamente
				IsLastChunk:   isLastChunk,
				ChunkSize:     len(chunkData),
				ChunkChecksum: "",                // TODO: implementar checksum MD5 si es necesario
				Timestamp:     time.Now().Unix(), // Unix timestamp
			}

			message := dto.WebSocketMessage{
				Type: "file_chunk",
				Data: chunk,
			}

			// Verificar que el cliente sigue conectado antes de cada chunk
			h.mutex.RLock()
			_, exists := h.pcConnections[transfer.TargetPCID()]
			h.mutex.RUnlock()

			if !exists {
				return fmt.Errorf("client disconnected during chunk transfer")
			}

			if err := clientConn.Conn.WriteJSON(message); err != nil {
				return fmt.Errorf("error sending chunk %d: %w", chunkIndex, err)
			}

			log.Printf("📦 Chunk %d/%d sent for transfer %s (size: %d bytes, last: %v)",
				chunkIndex+1, totalChunks, transfer.TransferID(), len(chunkData), isLastChunk)

			chunkIndex++ // Incrementar índice para próximo chunk

			// Pequeña pausa para no saturar la conexión
			time.Sleep(10 * time.Millisecond)

			return nil
		},
	)

	if err != nil {
		// Marcar transferencia como fallida
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		updateErr := h.fileTransferService.UpdateTransferStatus(
			ctx,
			transfer.TransferID(),
			filetransfer.TransferStatusFailed,
			fmt.Sprintf("Error enviando chunks: %v", err),
		)
		if updateErr != nil {
			log.Printf("Error updating transfer status after chunk failure: %v", updateErr)
		}

		return fmt.Errorf("error sending file chunks: %w", err)
	}

	log.Printf("✅ All chunks sent successfully for transfer %s (%d chunks)",
		transfer.TransferID(), totalChunks)

	return nil
}

// ProcessFileTransfer processes a complete file transfer from start to finish
func (h *WebSocketHandler) ProcessFileTransfer(transfer *filetransfer.FileTransfer) error {
	log.Printf("🚀 Starting file transfer process: %s (File: %s, Target: %s)",
		transfer.TransferID(), transfer.FileName(), transfer.TargetPCID())

	// 1. Enviar solicitud de transferencia al cliente
	if err := h.SendFileTransferRequestToClient(transfer); err != nil {
		return fmt.Errorf("error sending transfer request: %w", err)
	}

	// 2. Por ahora, para MVP, asumir que el cliente está listo y enviar chunks inmediatamente
	// En una implementación completa, esperaríamos una confirmación del cliente

	// Pequeña pausa para que el cliente procese la solicitud
	time.Sleep(100 * time.Millisecond)

	// 3. Enviar chunks del archivo
	if err := h.SendFileChunksToClient(transfer); err != nil {
		return fmt.Errorf("error sending file chunks: %w", err)
	}

	// 4. Marcar como completada (en una implementación completa, esto se haría cuando el cliente confirme)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.fileTransferService.UpdateTransferStatus(
		ctx,
		transfer.TransferID(),
		filetransfer.TransferStatusCompleted,
		"",
	)
	if err != nil {
		log.Printf("Error updating transfer status to COMPLETED: %v", err)
		return fmt.Errorf("error marking transfer as completed: %w", err)
	}

	log.Printf("🎉 File transfer completed successfully: %s", transfer.TransferID())

	// Notificar al administrador sobre la finalización exitosa
	if h.adminWSHandler != nil {
		// TODO: Implementar BroadcastFileTransferCompleted en AdminWebSocketHandler
		// h.adminWSHandler.BroadcastFileTransferCompleted(transfer.TransferID(), transfer.FileName(), transfer.TargetPCID())
	}

	return nil
}

// SendSessionEndedToClient notifica al cliente que una sesión ha terminado
func (h *WebSocketHandler) SendSessionEndedToClient(sessionID, clientPCID string) error {
	log.Printf("🔚 SESSION END: Attempting to send session ended notification to client PC: %s", clientPCID)

	h.mutex.RLock()
	clientConn, exists := h.pcConnections[clientPCID]
	h.mutex.RUnlock()

	if !exists {
		log.Printf("⚠️ SESSION END: Client PC %s not found in connections map", clientPCID)
		return nil // No es un error crítico si el cliente no está conectado
	}

	log.Printf("✅ SESSION END: Found client connection for PC: %s", clientPCID)

	// Crear mensaje de sesión terminada
	sessionEndedMsg := dto.WebSocketMessage{
		Type: "control_session_ended",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"reason":     "ended_by_admin",
			"message":    "Remote control session ended by administrator",
			"timestamp":  time.Now().Unix(),
		},
	}

	log.Printf("📡 SESSION END: Sending to client %s: %+v", clientPCID, sessionEndedMsg)

	// Enviar mensaje al cliente
	err := clientConn.Conn.WriteJSON(sessionEndedMsg)
	if err != nil {
		log.Printf("❌ SESSION END: Error sending to client %s: %v", clientPCID, err)
		return err
	}

	log.Printf("✅ SESSION END: Notification sent successfully to client %s", clientPCID)
	return nil
}

// processPendingTransfers procesa transferencias pendientes para un cliente recién conectado
func (h *WebSocketHandler) processPendingTransfers(clientPCID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("🔍 PENDING TRANSFERS: Iniciando búsqueda para cliente %s", clientPCID)

	// Obtener transferencias pendientes para este cliente
	transfers, err := h.fileTransferService.GetTransfersByTargetPC(ctx, clientPCID)
	if err != nil {
		log.Printf("❌ PENDING TRANSFERS: Error getting transfers for client %s: %v", clientPCID, err)
		return
	}

	log.Printf("📊 PENDING TRANSFERS: Encontradas %d transferencias totales para cliente %s", len(transfers), clientPCID)

	pendingTransfers := make([]*filetransfer.FileTransfer, 0)
	for i, transfer := range transfers {
		log.Printf("  [%d] Transfer ID: %s, Status: %s, File: %s", i+1, transfer.TransferID(), transfer.Status(), transfer.FileName())
		if transfer.Status() == filetransfer.TransferStatusPending {
			pendingTransfers = append(pendingTransfers, transfer)
		}
	}

	if len(pendingTransfers) > 0 {
		log.Printf("📋 PENDING TRANSFERS: Cliente %s se conectó con %d transferencias pendientes", clientPCID, len(pendingTransfers))

		// Procesar transferencias pendientes en una goroutine
		go func() {
			for i, transfer := range pendingTransfers {
				log.Printf("🔄 PENDING TRANSFERS: [%d/%d] Procesando transferencia pendiente: %s -> %s",
					i+1, len(pendingTransfers), transfer.FileName(), clientPCID)

				// Enviar solicitud de transferencia
				err := h.SendFileTransferRequestToClient(transfer)
				if err != nil {
					log.Printf("❌ PENDING TRANSFERS: Error sending transfer %s: %v", transfer.TransferID(), err)
					continue
				}

				// Esperar un poco entre transferencias
				time.Sleep(2 * time.Second)

				// Enviar chunks del archivo
				err = h.SendFileChunksToClient(transfer)
				if err != nil {
					log.Printf("❌ PENDING TRANSFERS: Error sending file chunks for transfer %s: %v", transfer.TransferID(), err)
				} else {
					log.Printf("✅ PENDING TRANSFERS: Transferencia %s procesada exitosamente", transfer.TransferID())
				}
			}
			log.Printf("🎉 PENDING TRANSFERS: Completado procesamiento de transferencias pendientes para cliente %s", clientPCID)
		}()
	} else {
		log.Printf("✅ PENDING TRANSFERS: No hay transferencias pendientes para cliente %s", clientPCID)
	}
}
