package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/videoservice"
)

// VideoHandler maneja las solicitudes HTTP relacionadas con videos y frames
type VideoHandler struct {
	sessionService *remotesessionservice.RemoteSessionService
	videoService   videoservice.IVideoService
	authService    *userservice.AuthService
}

// NewVideoHandler crea una nueva instancia del handler de video
func NewVideoHandler(
	sessionService *remotesessionservice.RemoteSessionService,
	videoService videoservice.IVideoService,
	authService *userservice.AuthService,
) *VideoHandler {
	return &VideoHandler{
		sessionService: sessionService,
		videoService:   videoService,
		authService:    authService,
	}
}

// GetRecordingMetadata obtiene los metadatos de una grabación por sessionId
// GET /api/admin/sessions/{sessionId}/recording/metadata
func (vh *VideoHandler) GetRecordingMetadata(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID requerido",
		})
		return
	}

	// Obtener videos por sessionID
	videos, err := vh.videoService.GetVideosBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Error obteniendo videos de la sesión",
		})
		return
	}

	if len(videos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No se encontraron grabaciones para esta sesión",
		})
		return
	}

	// Tomar el primer video (asumiendo una grabación por sesión)
	video := videos[0]

	// Calcular total de frames contando archivos en el directorio
	framesDir := video.FilePath() // Ahora contiene la ruta del directorio de frames
	totalFrames, err := vh.countFramesInDirectory(framesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Error contando frames de la grabación",
		})
		return
	}

	// Calcular FPS aproximado
	var fps float64
	if video.DurationSeconds() > 0 {
		fps = float64(totalFrames) / float64(video.DurationSeconds())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"video_id":         video.VideoID(),
			"session_id":       sessionID,
			"total_frames":     totalFrames,
			"duration_seconds": video.DurationSeconds(),
			"fps":              fps,
			"recorded_at":      video.RecordedAt(),
			"file_size_mb":     video.FileSizeMB(),
		},
	})
}

// GetVideoFrame sirve un frame individual de video
// GET /api/admin/sessions/{sessionId}/frames/{frameNumber}
func (vh *VideoHandler) GetVideoFrame(c *gin.Context) {
	sessionID := c.Param("sessionId")
	frameNumberStr := c.Param("frameNumber")

	if sessionID == "" || frameNumberStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID y número de frame requeridos",
		})
		return
	}

	frameNumber, err := strconv.Atoi(frameNumberStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Número de frame inválido",
		})
		return
	}

	// Obtener videos por sessionID
	videos, err := vh.videoService.GetVideosBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Error obteniendo videos de la sesión",
		})
		return
	}

	if len(videos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No se encontraron grabaciones para esta sesión",
		})
		return
	}

	// Tomar el primer video
	video := videos[0]

	// Construir ruta del frame
	frameFileName := fmt.Sprintf("frame_%06d.jpg", frameNumber)
	frameFilePath := filepath.Join(video.FilePath(), frameFileName)

	// Verificar que el archivo existe
	if _, err := os.Stat(frameFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Frame no encontrado",
		})
		return
	}

	// Servir el archivo JPEG con headers apropiados
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=3600") // Cache por 1 hora
	c.File(frameFilePath)
}

// countFramesInDirectory cuenta los archivos de frame en un directorio
func (vh *VideoHandler) countFramesInDirectory(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jpg" {
			count++
		}
	}

	return count, nil
}

// GetAllRecordings obtiene todas las grabaciones agrupadas por cliente
// GET /api/admin/recordings
func (vh *VideoHandler) GetAllRecordings(c *gin.Context) {
	// Obtener todas las sesiones que tienen videos
	sessions, err := vh.sessionService.GetAllSessionsWithVideos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Error obteniendo sesiones con videos",
		})
		return
	}

	// Agrupar por cliente con información adicional
	clientRecordings := make(map[string]interface{})
	processedClients := make(map[string]bool) // Para evitar consultas duplicadas

	for _, session := range sessions {
		clientPCID := session.ClientPCID()

		// Obtener videos de esta sesión
		videos, err := vh.videoService.GetVideosBySessionID(c.Request.Context(), session.SessionID())
		if err != nil {
			continue // Skip sessions with errors
		}

		if len(videos) == 0 {
			continue // Skip sessions without videos
		}

		// Crear estructura del cliente si no existe
		if clientRecordings[clientPCID] == nil {
			// Solo buscar info del cliente una vez
			var clientName string
			if !processedClients[clientPCID] {
				// Tomar solo los primeros 8 caracteres del UUID para mostrar
				shortID := clientPCID
				if len(clientPCID) > 8 {
					shortID = clientPCID[:8] + "..."
				}
				clientName = "Cliente " + shortID
				processedClients[clientPCID] = true
			}

			clientRecordings[clientPCID] = gin.H{
				"client_pc_id": clientPCID,
				"client_name":  clientName,
				"recordings":   []gin.H{},
			}
		}

		// Procesar cada video de la sesión
		for _, video := range videos {
			// Calcular total de frames
			framesDir := video.FilePath()
			totalFrames, frameErr := vh.countFramesInDirectory(framesDir)
			if frameErr != nil {
				totalFrames = 0
			}

			// Calcular FPS
			var fps float64
			if video.DurationSeconds() > 0 {
				fps = float64(totalFrames) / float64(video.DurationSeconds())
			}

			recording := gin.H{
				"video_id":         video.VideoID(),
				"session_id":       session.SessionID(),
				"recorded_at":      video.RecordedAt(),
				"duration_seconds": video.DurationSeconds(),
				"total_frames":     totalFrames,
				"fps":              fps,
				"file_size_mb":     video.FileSizeMB(),
				"session_status":   session.Status(),
			}

			// Agregar a la lista del cliente
			clientData := clientRecordings[clientPCID].(gin.H)
			recordings := clientData["recordings"].([]gin.H)
			clientData["recordings"] = append(recordings, recording)
		}
	}

	// Convertir mapa a slice para response JSON ordenado
	var result []interface{}
	for _, clientData := range clientRecordings {
		result = append(result, clientData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetClientRecordings obtiene las grabaciones de un cliente específico
// GET /api/admin/clients/{clientId}/recordings
func (vh *VideoHandler) GetClientRecordings(c *gin.Context) {
	clientPCID := c.Param("clientId")
	if clientPCID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Client ID requerido",
		})
		return
	}

	// Obtener sesiones de este cliente que tienen videos
	sessions, err := vh.sessionService.GetSessionsByClientPCID(c.Request.Context(), clientPCID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Error obteniendo sesiones del cliente",
		})
		return
	}

	var recordings []gin.H

	for _, session := range sessions {
		// Obtener videos de esta sesión
		videos, err := vh.videoService.GetVideosBySessionID(c.Request.Context(), session.SessionID())
		if err != nil {
			continue // Skip sessions with errors
		}

		for _, video := range videos {
			// Calcular total de frames
			framesDir := video.FilePath()
			totalFrames, frameErr := vh.countFramesInDirectory(framesDir)
			if frameErr != nil {
				totalFrames = 0
			}

			// Calcular FPS
			var fps float64
			if video.DurationSeconds() > 0 {
				fps = float64(totalFrames) / float64(video.DurationSeconds())
			}

			recording := gin.H{
				"video_id":         video.VideoID(),
				"session_id":       session.SessionID(),
				"recorded_at":      video.RecordedAt(),
				"duration_seconds": video.DurationSeconds(),
				"total_frames":     totalFrames,
				"fps":              fps,
				"file_size_mb":     video.FileSizeMB(),
				"session_status":   session.Status(),
			}

			recordings = append(recordings, recording)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"client_pc_id": clientPCID,
		"recordings":   recordings,
		"total_count":  len(recordings),
	})
}
