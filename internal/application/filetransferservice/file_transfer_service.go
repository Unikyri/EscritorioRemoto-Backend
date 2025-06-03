package filetransferservice

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/filetransfer"
)

// FileTransferService maneja la lógica de negocio para transferencias de archivos
type FileTransferService struct {
	fileTransferRepository interfaces.IFileTransferRepository
	actionLogRepository    interfaces.IActionLogRepository
	fileStorage            interfaces.IFileStorage
}

// NewFileTransferService crea una nueva instancia del servicio
func NewFileTransferService(
	fileTransferRepository interfaces.IFileTransferRepository,
	actionLogRepository interfaces.IActionLogRepository,
	fileStorage interfaces.IFileStorage,
) *FileTransferService {
	return &FileTransferService{
		fileTransferRepository: fileTransferRepository,
		actionLogRepository:    actionLogRepository,
		fileStorage:            fileStorage,
	}
}

// InitiateServerToClientTransferRequest representa la solicitud de transferencia
type InitiateServerToClientTransferRequest struct {
	AdminUserID    string
	SessionID      string
	TargetPCID     string
	ServerFilePath string
	ClientFileName string
}

// InitiateServerToClientTransfer inicia una transferencia de archivo del servidor al cliente
func (s *FileTransferService) InitiateServerToClientTransfer(
	ctx context.Context,
	req InitiateServerToClientTransferRequest,
) (*filetransfer.FileTransfer, error) {
	// 1. Verificar que el archivo del servidor existe y es accesible
	fileInfo, err := s.validateServerFile(req.ServerFilePath)
	if err != nil {
		return nil, fmt.Errorf("archivo del servidor no válido: %w", err)
	}

	// 2. Calcular tamaño del archivo en MB
	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	// 3. Definir ruta de destino en el cliente (predefinida para MVP)
	destinationPath := s.getClientDestinationPath(req.ClientFileName)

	// 4. Crear registro FileTransfer en BD con estado PENDING
	transfer := filetransfer.NewFileTransfer(
		req.ClientFileName,
		req.ServerFilePath,
		destinationPath,
		req.SessionID,
		req.AdminUserID,
		req.TargetPCID,
		fileSizeMB,
	)

	err = s.fileTransferRepository.Save(ctx, transfer)
	if err != nil {
		return nil, fmt.Errorf("error guardando transferencia: %w", err)
	}

	// 5. Registrar inicio de transferencia en ActionLog
	err = s.logTransferAction(ctx, req.AdminUserID, transfer.TransferID(), "FILE_TRANSFER_INITIATED",
		fmt.Sprintf("Iniciada transferencia de archivo %s a PC %s", req.ClientFileName, req.TargetPCID))
	if err != nil {
		// Log error but don't fail the transfer
		fmt.Printf("Error logging transfer action: %v\n", err)
	}

	return transfer, nil
}

// UpdateTransferStatus actualiza el estado de una transferencia
func (s *FileTransferService) UpdateTransferStatus(
	ctx context.Context,
	transferID string,
	status filetransfer.TransferStatus,
	errorMessage string,
) error {
	err := s.fileTransferRepository.UpdateStatus(ctx, transferID, status, errorMessage)
	if err != nil {
		return fmt.Errorf("error actualizando estado de transferencia: %w", err)
	}

	// Log the status change
	var actionType string
	switch status {
	case filetransfer.TransferStatusInProgress:
		actionType = "FILE_TRANSFER_STARTED"
	case filetransfer.TransferStatusCompleted:
		actionType = "FILE_TRANSFER_COMPLETED"
	case filetransfer.TransferStatusFailed:
		actionType = "FILE_TRANSFER_FAILED"
	}

	if actionType != "" {
		// Get transfer to log details
		transfer, err := s.fileTransferRepository.FindByID(ctx, transferID)
		if err == nil {
			description := fmt.Sprintf("Transferencia %s - Estado: %s", transfer.FileName(), status)
			if errorMessage != "" {
				description += fmt.Sprintf(" - Error: %s", errorMessage)
			}

			s.logTransferAction(ctx, transfer.InitiatingUserID(), transferID, actionType, description)
		}
	}

	return nil
}

// ReadFileInChunks lee un archivo en chunks para transferencia
func (s *FileTransferService) ReadFileInChunks(filePath string, chunkSize int, callback func([]byte, bool) error) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error leyendo archivo: %w", err)
		}

		if n == 0 {
			break
		}

		isLastChunk := err == io.EOF
		chunk := buffer[:n]

		if err := callback(chunk, isLastChunk); err != nil {
			return fmt.Errorf("error procesando chunk: %w", err)
		}

		if isLastChunk {
			break
		}
	}

	return nil
}

// GetTransferByID obtiene una transferencia por su ID
func (s *FileTransferService) GetTransferByID(ctx context.Context, transferID string) (*filetransfer.FileTransfer, error) {
	return s.fileTransferRepository.FindByID(ctx, transferID)
}

// GetTransfersBySessionID obtiene todas las transferencias de una sesión
func (s *FileTransferService) GetTransfersBySessionID(ctx context.Context, sessionID string) ([]*filetransfer.FileTransfer, error) {
	return s.fileTransferRepository.FindBySessionID(ctx, sessionID)
}

// GetPendingTransfers obtiene todas las transferencias pendientes
func (s *FileTransferService) GetPendingTransfers(ctx context.Context) ([]*filetransfer.FileTransfer, error) {
	return s.fileTransferRepository.FindPendingTransfers(ctx)
}

// GetTransfersByTargetPC obtiene todas las transferencias enviadas a un PC específico
func (s *FileTransferService) GetTransfersByTargetPC(ctx context.Context, targetPCID string) ([]*filetransfer.FileTransfer, error) {
	return s.fileTransferRepository.FindByTargetPCID(ctx, targetPCID)
}

// validateServerFile valida que el archivo del servidor existe y es accesible
func (s *FileTransferService) validateServerFile(filePath string) (os.FileInfo, error) {
	// Verificar que el archivo existe
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("archivo no encontrado: %s", filePath)
		}
		return nil, fmt.Errorf("error accediendo al archivo: %w", err)
	}

	// Verificar que es un archivo regular (no directorio)
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("la ruta especificada es un directorio, no un archivo: %s", filePath)
	}

	// Verificar que el archivo es legible
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("archivo no es legible: %w", err)
	}
	file.Close()

	return fileInfo, nil
}

// getClientDestinationPath genera la ruta de destino en el cliente (predefinida para MVP)
func (s *FileTransferService) getClientDestinationPath(fileName string) string {
	// Para MVP, usar una ruta consistente con la configuración del cliente
	// El cliente detecta automáticamente Descargas/Downloads y usa RemoteDesk como subcarpeta
	return filepath.Join("Descargas", "RemoteDesk", fileName)
}

// logTransferAction registra una acción de transferencia en el log de auditoría
func (s *FileTransferService) logTransferAction(ctx context.Context, userID, entityID, actionType, description string) error {
	// Import and use the action log service if available
	// For now, just return nil to avoid circular dependencies
	// This should be properly implemented when action log is needed
	return nil
}
