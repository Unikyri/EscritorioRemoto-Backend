package handlers

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/filetransferservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/filetransfer"
)

// WebSocketHandlerInterface define los métodos que necesitamos del WebSocketHandler
type WebSocketHandlerInterface interface {
	ProcessFileTransfer(transfer *filetransfer.FileTransfer) error
}

// FileTransferHandler maneja las solicitudes HTTP de transferencia de archivos
type FileTransferHandler struct {
	fileTransferService *filetransferservice.FileTransferService
	authService         *userservice.AuthService
	fileStorage         interfaces.IFileStorage
	webSocketHandler    WebSocketHandlerInterface
}

// NewFileTransferHandler crea una nueva instancia del handler
func NewFileTransferHandler(
	fileTransferService *filetransferservice.FileTransferService,
	authService *userservice.AuthService,
	fileStorage interfaces.IFileStorage,
	webSocketHandler WebSocketHandlerInterface,
) *FileTransferHandler {
	return &FileTransferHandler{
		fileTransferService: fileTransferService,
		authService:         authService,
		fileStorage:         fileStorage,
		webSocketHandler:    webSocketHandler,
	}
}

// SendFileRequest estructura de la solicitud de envío de archivo
type SendFileRequest struct {
	TargetPCID     string `json:"target_pc_id" binding:"required"`
	ClientFileName string `json:"client_file_name" binding:"required"`
	ServerFilePath string `json:"server_file_path,omitempty"` // Opcional si se sube archivo
}

// SendFile maneja el endpoint POST /api/admin/sessions/{sessionID}/files/send
func (h *FileTransferHandler) SendFile(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID requerido",
		})
		return
	}

	// Obtener usuario autenticado
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Usuario no autenticado",
		})
		return
	}

	adminUserID := userClaims.(*userservice.JWTClaims).UserID

	var serverFilePath string
	var request SendFileRequest

	// Verificar si hay un archivo en el form (multipart)
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		// Hay un archivo subido
		defer file.Close()

		// Obtener otros campos del form
		request.TargetPCID = c.PostForm("target_pc_id")
		request.ClientFileName = c.PostForm("client_file_name")

		if request.TargetPCID == "" || request.ClientFileName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "target_pc_id y client_file_name son requeridos",
			})
			return
		}

		// Guardar archivo temporalmente en el servidor
		tempPath, err := h.saveUploadedFile(c, file, header, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Error guardando archivo: %v", err),
			})
			return
		}
		serverFilePath = tempPath

	} else {
		// No hay archivo subido, usar JSON con ruta de archivo existente
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Datos de solicitud inválidos: %v", err),
			})
			return
		}

		if request.ServerFilePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Se requiere server_file_path o subir un archivo",
			})
			return
		}
		serverFilePath = request.ServerFilePath
	}

	// Iniciar transferencia
	transferRequest := filetransferservice.InitiateServerToClientTransferRequest{
		AdminUserID:    adminUserID,
		SessionID:      sessionID,
		TargetPCID:     request.TargetPCID,
		ServerFilePath: serverFilePath,
		ClientFileName: request.ClientFileName,
	}

	transfer, err := h.fileTransferService.InitiateServerToClientTransfer(c.Request.Context(), transferRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Error iniciando transferencia: %v", err),
		})
		return
	}

	// 🚀 PROCESAR TRANSFERENCIA INMEDIATAMENTE
	// Procesar la transferencia en una goroutine para no bloquear la respuesta HTTP
	if h.webSocketHandler != nil {
		go func() {
			log.Printf("🔄 AUTO-PROCESSING: Iniciando procesamiento automático de transferencia %s", transfer.TransferID())
			err := h.webSocketHandler.ProcessFileTransfer(transfer)
			if err != nil {
				log.Printf("❌ AUTO-PROCESSING: Error procesando transferencia %s: %v", transfer.TransferID(), err)
			} else {
				log.Printf("✅ AUTO-PROCESSING: Transferencia %s procesada exitosamente", transfer.TransferID())
			}
		}()
	} else {
		log.Printf("⚠️ AUTO-PROCESSING: WebSocketHandler no disponible, transferencia %s quedará pendiente", transfer.TransferID())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transferencia iniciada exitosamente",
		"data": gin.H{
			"transfer_id":      transfer.TransferID(),
			"file_name":        transfer.FileName(),
			"target_pc_id":     transfer.TargetPCID(),
			"session_id":       transfer.AssociatedSessionID(),
			"status":           transfer.Status(),
			"file_size_mb":     transfer.FileSizeMB(),
			"destination_path": transfer.DestinationPathClient(),
		},
	})
}

// GetTransfersBySession obtiene todas las transferencias de una sesión
func (h *FileTransferHandler) GetTransfersBySession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID requerido",
		})
		return
	}

	transfers, err := h.fileTransferService.GetTransfersBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Error obteniendo transferencias: %v", err),
		})
		return
	}

	var transferData []gin.H
	for _, transfer := range transfers {
		transferData = append(transferData, gin.H{
			"transfer_id":      transfer.TransferID(),
			"file_name":        transfer.FileName(),
			"target_pc_id":     transfer.TargetPCID(),
			"session_id":       transfer.AssociatedSessionID(),
			"status":           transfer.Status(),
			"file_size_mb":     transfer.FileSizeMB(),
			"destination_path": transfer.DestinationPathClient(),
			"transfer_time":    transfer.TransferTime(),
			"error_message":    transfer.ErrorMessage(),
			"created_at":       transfer.CreatedAt(),
			"updated_at":       transfer.UpdatedAt(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transferData,
		"count":   len(transferData),
	})
}

// GetTransferStatus obtiene el estado de una transferencia específica
func (h *FileTransferHandler) GetTransferStatus(c *gin.Context) {
	transferID := c.Param("transferId")
	if transferID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Transfer ID requerido",
		})
		return
	}

	transfer, err := h.fileTransferService.GetTransferByID(c.Request.Context(), transferID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Transferencia no encontrada: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"transfer_id":      transfer.TransferID(),
			"file_name":        transfer.FileName(),
			"target_pc_id":     transfer.TargetPCID(),
			"session_id":       transfer.AssociatedSessionID(),
			"status":           transfer.Status(),
			"file_size_mb":     transfer.FileSizeMB(),
			"destination_path": transfer.DestinationPathClient(),
			"transfer_time":    transfer.TransferTime(),
			"error_message":    transfer.ErrorMessage(),
			"created_at":       transfer.CreatedAt(),
			"updated_at":       transfer.UpdatedAt(),
		},
	})
}

// saveUploadedFile guarda un archivo subido temporalmente en el servidor
func (h *FileTransferHandler) saveUploadedFile(c *gin.Context, file interface{}, header interface{}, sessionID string) (string, error) {
	fileHeader := header.(*multipart.FileHeader)
	multipartFile := file.(multipart.File)

	// Crear directorio de destino para transferencias
	destDir := filepath.Join("file_transfers", sessionID)
	filename := fileHeader.Filename
	relativePath := filepath.Join(destDir, filename)

	// Usar el servicio de almacenamiento si está disponible
	if h.fileStorage != nil {
		// Leer contenido del archivo
		content, err := io.ReadAll(multipartFile)
		if err != nil {
			return "", fmt.Errorf("error leyendo archivo subido: %w", err)
		}

		// Guardar usando el storage service
		savedPath, err := h.fileStorage.SaveFile(c.Request.Context(), relativePath, content)
		if err != nil {
			return "", fmt.Errorf("error guardando archivo: %w", err)
		}

		return savedPath, nil
	}

	// Fallback: usar método simple de guardado local
	tempDir := filepath.Join("temp", "file_transfers", sessionID)
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("error creando directorio temporal: %w", err)
	}

	filePath := filepath.Join(tempDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creando archivo: %w", err)
	}
	defer dst.Close()

	// Copiar contenido del archivo subido
	_, err = io.Copy(dst, multipartFile)
	if err != nil {
		return "", fmt.Errorf("error copiando archivo: %w", err)
	}

	return filePath, nil
}

// GetPendingTransfers obtiene todas las transferencias pendientes
func (h *FileTransferHandler) GetPendingTransfers(c *gin.Context) {
	transfers, err := h.fileTransferService.GetPendingTransfers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Error obteniendo transferencias pendientes: %v", err),
		})
		return
	}

	var transferData []gin.H
	for _, transfer := range transfers {
		transferData = append(transferData, gin.H{
			"transfer_id":      transfer.TransferID(),
			"file_name":        transfer.FileName(),
			"target_pc_id":     transfer.TargetPCID(),
			"session_id":       transfer.AssociatedSessionID(),
			"status":           transfer.Status(),
			"file_size_mb":     transfer.FileSizeMB(),
			"destination_path": transfer.DestinationPathClient(),
			"transfer_time":    transfer.TransferTime(),
			"created_at":       transfer.CreatedAt(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transferData,
		"count":   len(transferData),
	})
}

// GetTransfersByClient obtiene todas las transferencias de un cliente específico
func (h *FileTransferHandler) GetTransfersByClient(c *gin.Context) {
	clientPCID := c.Param("clientId")
	if clientPCID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Client PC ID requerido",
		})
		return
	}

	transfers, err := h.fileTransferService.GetTransfersByTargetPC(c.Request.Context(), clientPCID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Error obteniendo transferencias del cliente: %v", err),
		})
		return
	}

	var transferData []gin.H
	for _, transfer := range transfers {
		transferData = append(transferData, gin.H{
			"transfer_id":      transfer.TransferID(),
			"file_name":        transfer.FileName(),
			"target_pc_id":     transfer.TargetPCID(),
			"session_id":       transfer.AssociatedSessionID(),
			"status":           transfer.Status(),
			"file_size_mb":     transfer.FileSizeMB(),
			"destination_path": transfer.DestinationPathClient(),
			"transfer_time":    transfer.TransferTime(),
			"error_message":    transfer.ErrorMessage(),
			"created_at":       transfer.CreatedAt(),
			"updated_at":       transfer.UpdatedAt(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transferData,
		"count":   len(transferData),
	})
}
