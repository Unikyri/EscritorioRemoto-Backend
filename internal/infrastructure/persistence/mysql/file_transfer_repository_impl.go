package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/filetransfer"
)

// FileTransferRepositoryImpl implementa IFileTransferRepository para MySQL
type FileTransferRepositoryImpl struct {
	db *sql.DB
}

// NewFileTransferRepository crea una nueva instancia del repositorio
func NewFileTransferRepository(db *sql.DB) *FileTransferRepositoryImpl {
	return &FileTransferRepositoryImpl{
		db: db,
	}
}

// Save guarda una nueva transferencia de archivo en la base de datos
func (r *FileTransferRepositoryImpl) Save(ctx context.Context, transfer *filetransfer.FileTransfer) error {
	query := `
		INSERT INTO file_transfers (
			transfer_id, file_name, source_path_server, destination_path_client,
			transfer_time, status, associated_session_id, initiating_user_id,
			target_pc_id, file_size_mb, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		transfer.TransferID(),
		transfer.FileName(),
		transfer.SourcePathServer(),
		transfer.DestinationPathClient(),
		transfer.TransferTime(),
		string(transfer.Status()),
		transfer.AssociatedSessionID(),
		transfer.InitiatingUserID(),
		transfer.TargetPCID(),
		transfer.FileSizeMB(),
		transfer.CreatedAt(),
		transfer.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("error guardando transferencia de archivo: %w", err)
	}

	return nil
}

// UpdateStatus actualiza el estado de una transferencia existente
func (r *FileTransferRepositoryImpl) UpdateStatus(
	ctx context.Context,
	transferID string,
	status filetransfer.TransferStatus,
	errorMessage string,
) error {
	query := `
		UPDATE file_transfers 
		SET status = ?, updated_at = ?
		WHERE transfer_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(status), time.Now(), transferID)
	if err != nil {
		return fmt.Errorf("error actualizando estado de transferencia: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verificando filas afectadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transferencia no encontrada: %s", transferID)
	}

	return nil
}

// FindByID busca una transferencia por su ID
func (r *FileTransferRepositoryImpl) FindByID(ctx context.Context, transferID string) (*filetransfer.FileTransfer, error) {
	query := `
		SELECT transfer_id, file_name, source_path_server, destination_path_client,
			   transfer_time, status, associated_session_id, initiating_user_id,
			   target_pc_id, file_size_mb, created_at, updated_at
		FROM file_transfers
		WHERE transfer_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, transferID)
	return r.scanFileTransfer(row)
}

// FindBySessionID busca todas las transferencias asociadas a una sesión
func (r *FileTransferRepositoryImpl) FindBySessionID(ctx context.Context, sessionID string) ([]*filetransfer.FileTransfer, error) {
	query := `
		SELECT transfer_id, file_name, source_path_server, destination_path_client,
			   transfer_time, status, associated_session_id, initiating_user_id,
			   target_pc_id, file_size_mb, created_at, updated_at
		FROM file_transfers
		WHERE associated_session_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("error consultando transferencias por sesión: %w", err)
	}
	defer rows.Close()

	return r.scanFileTransfers(rows)
}

// FindByTargetPCID busca todas las transferencias enviadas a un PC específico
func (r *FileTransferRepositoryImpl) FindByTargetPCID(ctx context.Context, targetPCID string) ([]*filetransfer.FileTransfer, error) {
	query := `
		SELECT transfer_id, file_name, source_path_server, destination_path_client,
			   transfer_time, status, associated_session_id, initiating_user_id,
			   target_pc_id, file_size_mb, created_at, updated_at
		FROM file_transfers
		WHERE target_pc_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, targetPCID)
	if err != nil {
		return nil, fmt.Errorf("error consultando transferencias por PC: %w", err)
	}
	defer rows.Close()

	return r.scanFileTransfers(rows)
}

// FindByInitiatingUserID busca todas las transferencias iniciadas por un usuario
func (r *FileTransferRepositoryImpl) FindByInitiatingUserID(ctx context.Context, userID string) ([]*filetransfer.FileTransfer, error) {
	query := `
		SELECT transfer_id, file_name, source_path_server, destination_path_client,
			   transfer_time, status, associated_session_id, initiating_user_id,
			   target_pc_id, file_size_mb, created_at, updated_at
		FROM file_transfers
		WHERE initiating_user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error consultando transferencias por usuario: %w", err)
	}
	defer rows.Close()

	return r.scanFileTransfers(rows)
}

// FindPendingTransfers busca todas las transferencias en estado PENDING
func (r *FileTransferRepositoryImpl) FindPendingTransfers(ctx context.Context) ([]*filetransfer.FileTransfer, error) {
	return r.findByStatus(ctx, filetransfer.TransferStatusPending)
}

// FindInProgressTransfers busca todas las transferencias en estado IN_PROGRESS
func (r *FileTransferRepositoryImpl) FindInProgressTransfers(ctx context.Context) ([]*filetransfer.FileTransfer, error) {
	return r.findByStatus(ctx, filetransfer.TransferStatusInProgress)
}

// findByStatus busca transferencias por estado
func (r *FileTransferRepositoryImpl) findByStatus(ctx context.Context, status filetransfer.TransferStatus) ([]*filetransfer.FileTransfer, error) {
	query := `
		SELECT transfer_id, file_name, source_path_server, destination_path_client,
			   transfer_time, status, associated_session_id, initiating_user_id,
			   target_pc_id, file_size_mb, created_at, updated_at
		FROM file_transfers
		WHERE status = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, string(status))
	if err != nil {
		return nil, fmt.Errorf("error consultando transferencias por estado: %w", err)
	}
	defer rows.Close()

	return r.scanFileTransfers(rows)
}

// scanFileTransfer convierte una fila de BD en una entidad FileTransfer
func (r *FileTransferRepositoryImpl) scanFileTransfer(row *sql.Row) (*filetransfer.FileTransfer, error) {
	var transferID, fileName, sourcePathServer, destinationPathClient string
	var transferTime time.Time
	var statusStr, associatedSessionID, initiatingUserID, targetPCID string
	var fileSizeMB float64
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&transferID, &fileName, &sourcePathServer, &destinationPathClient,
		&transferTime, &statusStr, &associatedSessionID, &initiatingUserID,
		&targetPCID, &fileSizeMB, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transferencia no encontrada")
		}
		return nil, fmt.Errorf("error escaneando transferencia: %w", err)
	}

	// Usar el constructor para hidratación desde BD
	transfer := filetransfer.NewFileTransferFromDB(
		transferID,
		fileName,
		sourcePathServer,
		destinationPathClient,
		transferTime,
		filetransfer.TransferStatus(statusStr),
		associatedSessionID,
		initiatingUserID,
		targetPCID,
		fileSizeMB,
		"", // errorMessage (no está en esta consulta, se agregará si es necesario)
		createdAt,
		updatedAt,
	)

	return transfer, nil
}

// scanFileTransfers convierte múltiples filas de BD en entidades FileTransfer
func (r *FileTransferRepositoryImpl) scanFileTransfers(rows *sql.Rows) ([]*filetransfer.FileTransfer, error) {
	var transfers []*filetransfer.FileTransfer

	for rows.Next() {
		var transferID, fileName, sourcePathServer, destinationPathClient string
		var transferTime time.Time
		var statusStr, associatedSessionID, initiatingUserID, targetPCID string
		var fileSizeMB float64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&transferID, &fileName, &sourcePathServer, &destinationPathClient,
			&transferTime, &statusStr, &associatedSessionID, &initiatingUserID,
			&targetPCID, &fileSizeMB, &createdAt, &updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("error escaneando transferencia: %w", err)
		}

		// Usar el constructor para hidratación desde BD
		transfer := filetransfer.NewFileTransferFromDB(
			transferID,
			fileName,
			sourcePathServer,
			destinationPathClient,
			transferTime,
			filetransfer.TransferStatus(statusStr),
			associatedSessionID,
			initiatingUserID,
			targetPCID,
			fileSizeMB,
			"", // errorMessage (no está en esta consulta, se agregará si es necesario)
			createdAt,
			updatedAt,
		)

		transfers = append(transfers, transfer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando transferencias: %w", err)
	}

	return transfers, nil
} 