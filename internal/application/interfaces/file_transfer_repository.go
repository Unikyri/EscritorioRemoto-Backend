package interfaces

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/filetransfer"
)

// IFileTransferRepository define la interfaz para el repositorio de transferencias de archivos
type IFileTransferRepository interface {
	// Save guarda una nueva transferencia de archivo en la base de datos
	Save(ctx context.Context, transfer *filetransfer.FileTransfer) error

	// UpdateStatus actualiza el estado de una transferencia existente
	UpdateStatus(ctx context.Context, transferID string, status filetransfer.TransferStatus, errorMessage string) error

	// FindByID busca una transferencia por su ID
	FindByID(ctx context.Context, transferID string) (*filetransfer.FileTransfer, error)

	// FindBySessionID busca todas las transferencias asociadas a una sesión
	FindBySessionID(ctx context.Context, sessionID string) ([]*filetransfer.FileTransfer, error)

	// FindByTargetPCID busca todas las transferencias enviadas a un PC específico
	FindByTargetPCID(ctx context.Context, targetPCID string) ([]*filetransfer.FileTransfer, error)

	// FindByInitiatingUserID busca todas las transferencias iniciadas por un usuario
	FindByInitiatingUserID(ctx context.Context, userID string) ([]*filetransfer.FileTransfer, error)

	// FindPendingTransfers busca todas las transferencias en estado PENDING
	FindPendingTransfers(ctx context.Context) ([]*filetransfer.FileTransfer, error)

	// FindInProgressTransfers busca todas las transferencias en estado IN_PROGRESS
	FindInProgressTransfers(ctx context.Context) ([]*filetransfer.FileTransfer, error)
} 