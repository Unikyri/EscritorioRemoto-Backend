package filetransfer

import (
	"time"

	"github.com/google/uuid"
)

// TransferStatus representa los posibles estados de una transferencia de archivo
type TransferStatus string

const (
	TransferStatusPending    TransferStatus = "PENDING"
	TransferStatusInProgress TransferStatus = "IN_PROGRESS"
	TransferStatusCompleted  TransferStatus = "COMPLETED"
	TransferStatusFailed     TransferStatus = "FAILED"
)

// FileTransfer representa una transferencia de archivo del servidor al cliente
type FileTransfer struct {
	transferID           string
	fileName             string
	sourcePathServer     string
	destinationPathClient string
	transferTime         time.Time
	status               TransferStatus
	associatedSessionID  string
	initiatingUserID     string
	targetPCID          string
	fileSizeMB          float64
	errorMessage        string
	createdAt           time.Time
	updatedAt           time.Time
}

// NewFileTransfer crea una nueva instancia de FileTransfer
func NewFileTransfer(
	fileName string,
	sourcePathServer string,
	destinationPathClient string,
	associatedSessionID string,
	initiatingUserID string,
	targetPCID string,
	fileSizeMB float64,
) *FileTransfer {
	return &FileTransfer{
		transferID:           uuid.New().String(),
		fileName:             fileName,
		sourcePathServer:     sourcePathServer,
		destinationPathClient: destinationPathClient,
		transferTime:         time.Now(),
		status:               TransferStatusPending,
		associatedSessionID:  associatedSessionID,
		initiatingUserID:     initiatingUserID,
		targetPCID:          targetPCID,
		fileSizeMB:          fileSizeMB,
		createdAt:           time.Now(),
		updatedAt:           time.Now(),
	}
}

// NewFileTransferFromDB crea una instancia de FileTransfer desde datos de BD
func NewFileTransferFromDB(
	transferID string,
	fileName string,
	sourcePathServer string,
	destinationPathClient string,
	transferTime time.Time,
	status TransferStatus,
	associatedSessionID string,
	initiatingUserID string,
	targetPCID string,
	fileSizeMB float64,
	errorMessage string,
	createdAt time.Time,
	updatedAt time.Time,
) *FileTransfer {
	return &FileTransfer{
		transferID:           transferID,
		fileName:             fileName,
		sourcePathServer:     sourcePathServer,
		destinationPathClient: destinationPathClient,
		transferTime:         transferTime,
		status:               status,
		associatedSessionID:  associatedSessionID,
		initiatingUserID:     initiatingUserID,
		targetPCID:          targetPCID,
		fileSizeMB:          fileSizeMB,
		errorMessage:        errorMessage,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}
}

// Getters
func (ft *FileTransfer) TransferID() string           { return ft.transferID }
func (ft *FileTransfer) FileName() string            { return ft.fileName }
func (ft *FileTransfer) SourcePathServer() string    { return ft.sourcePathServer }
func (ft *FileTransfer) DestinationPathClient() string { return ft.destinationPathClient }
func (ft *FileTransfer) TransferTime() time.Time     { return ft.transferTime }
func (ft *FileTransfer) Status() TransferStatus      { return ft.status }
func (ft *FileTransfer) AssociatedSessionID() string { return ft.associatedSessionID }
func (ft *FileTransfer) InitiatingUserID() string    { return ft.initiatingUserID }
func (ft *FileTransfer) TargetPCID() string         { return ft.targetPCID }
func (ft *FileTransfer) FileSizeMB() float64         { return ft.fileSizeMB }
func (ft *FileTransfer) ErrorMessage() string        { return ft.errorMessage }
func (ft *FileTransfer) CreatedAt() time.Time        { return ft.createdAt }
func (ft *FileTransfer) UpdatedAt() time.Time        { return ft.updatedAt }

// UpdateStatus actualiza el estado de la transferencia
func (ft *FileTransfer) UpdateStatus(status TransferStatus, errorMessage string) {
	ft.status = status
	ft.errorMessage = errorMessage
	ft.updatedAt = time.Now()
}

// SetInProgress marca la transferencia como en progreso
func (ft *FileTransfer) SetInProgress() {
	ft.UpdateStatus(TransferStatusInProgress, "")
}

// SetCompleted marca la transferencia como completada
func (ft *FileTransfer) SetCompleted() {
	ft.UpdateStatus(TransferStatusCompleted, "")
}

// SetFailed marca la transferencia como fallida
func (ft *FileTransfer) SetFailed(errorMessage string) {
	ft.UpdateStatus(TransferStatusFailed, errorMessage)
}

// IsCompleted verifica si la transferencia está completada
func (ft *FileTransfer) IsCompleted() bool {
	return ft.status == TransferStatusCompleted
}

// IsFailed verifica si la transferencia falló
func (ft *FileTransfer) IsFailed() bool {
	return ft.status == TransferStatusFailed
}

// IsInProgress verifica si la transferencia está en progreso
func (ft *FileTransfer) IsInProgress() bool {
	return ft.status == TransferStatusInProgress
}

// IsPending verifica si la transferencia está pendiente
func (ft *FileTransfer) IsPending() bool {
	return ft.status == TransferStatusPending
} 