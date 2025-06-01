package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// MySQLClientPCRepository implements IClientPCRepository using MySQL
type MySQLClientPCRepository struct {
	db *sql.DB
}

// NewMySQLClientPCRepository creates a new instance of MySQLClientPCRepository
func NewMySQLClientPCRepository(db *sql.DB) interfaces.IClientPCRepository {
	return &MySQLClientPCRepository{
		db: db,
	}
}

// Save stores a new ClientPC or updates an existing one
func (r *MySQLClientPCRepository) Save(ctx context.Context, pc *clientpc.ClientPC) error {
	// First, try to update existing record
	updateQuery := `
		UPDATE client_pcs 
		SET ip = ?, connection_status = ?, last_seen_at = ?, updated_at = ?
		WHERE pc_id = ?`

	result, err := r.db.ExecContext(ctx, updateQuery,
		pc.IP,
		pc.ConnectionStatus.String(),
		pc.LastSeenAt,
		pc.UpdatedAt,
		pc.PCID,
	)

	if err != nil {
		return fmt.Errorf("error updating ClientPC: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	// If no rows were updated, insert new record
	if rowsAffected == 0 {
		insertQuery := `
			INSERT INTO client_pcs (pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = r.db.ExecContext(ctx, insertQuery,
			pc.PCID,
			pc.Identifier,
			pc.IP,
			pc.ConnectionStatus.String(),
			pc.RegisteredAt,
			pc.OwnerUserID,
			pc.LastSeenAt,
			pc.CreatedAt,
			pc.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("error inserting new ClientPC: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a ClientPC by its ID
func (r *MySQLClientPCRepository) FindByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE pc_id = ?`

	row := r.db.QueryRowContext(ctx, query, pcID)

	pc, err := r.scanClientPC(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // PC not found
		}
		return nil, fmt.Errorf("error finding ClientPC by ID: %w", err)
	}

	return pc, nil
}

// FindByIdentifierAndOwner retrieves a ClientPC by identifier and owner user ID
func (r *MySQLClientPCRepository) FindByIdentifierAndOwner(ctx context.Context, identifier string, ownerID string) (*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE identifier = ? AND owner_user_id = ?`

	row := r.db.QueryRowContext(ctx, query, identifier, ownerID)

	pc, err := r.scanClientPC(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // PC not found
		}
		return nil, fmt.Errorf("error finding ClientPC by identifier and owner: %w", err)
	}

	return pc, nil
}

// FindByOwner retrieves all ClientPCs belonging to a specific owner
func (r *MySQLClientPCRepository) FindByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE owner_user_id = ?
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error finding ClientPCs by owner: %w", err)
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// FindOnlineByOwner retrieves all online ClientPCs belonging to a specific owner
func (r *MySQLClientPCRepository) FindOnlineByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE owner_user_id = ? AND connection_status = 'ONLINE'
		ORDER BY last_seen_at DESC`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error finding online ClientPCs by owner: %w", err)
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// UpdateConnectionStatus updates the connection status of a ClientPC
func (r *MySQLClientPCRepository) UpdateConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error {
	query := `
		UPDATE client_pcs 
		SET connection_status = ?, updated_at = ?
		WHERE pc_id = ?`

	result, err := r.db.ExecContext(ctx, query, status.String(), time.Now(), pcID)
	if err != nil {
		return fmt.Errorf("error updating connection status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no ClientPC found with ID: %s", pcID)
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp of a ClientPC
func (r *MySQLClientPCRepository) UpdateLastSeen(ctx context.Context, pcID string) error {
	query := `
		UPDATE client_pcs 
		SET last_seen_at = ?, updated_at = ?
		WHERE pc_id = ?`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, now, pcID)
	if err != nil {
		return fmt.Errorf("error updating last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no ClientPC found with ID: %s", pcID)
	}

	return nil
}

// Delete removes a ClientPC from the repository
func (r *MySQLClientPCRepository) Delete(ctx context.Context, pcID string) error {
	query := `DELETE FROM client_pcs WHERE pc_id = ?`

	result, err := r.db.ExecContext(ctx, query, pcID)
	if err != nil {
		return fmt.Errorf("error deleting ClientPC: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no ClientPC found with ID: %s", pcID)
	}

	return nil
}

// FindAll retrieves all ClientPCs with optional filtering
func (r *MySQLClientPCRepository) FindAll(ctx context.Context, limit, offset int) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id, last_seen_at, created_at, updated_at
		FROM client_pcs 
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error finding all ClientPCs: %w", err)
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// CountByOwner returns the count of ClientPCs for a specific owner
func (r *MySQLClientPCRepository) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	query := `SELECT COUNT(*) FROM client_pcs WHERE owner_user_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, ownerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting ClientPCs by owner: %w", err)
	}

	return count, nil
}

// Helper methods for scanning results

// scanClientPC scans a single row into a ClientPC struct
func (r *MySQLClientPCRepository) scanClientPC(row *sql.Row) (*clientpc.ClientPC, error) {
	var pc clientpc.ClientPC
	var connectionStatusStr string
	var lastSeenAt sql.NullTime

	err := row.Scan(
		&pc.PCID,
		&pc.Identifier,
		&pc.IP,
		&connectionStatusStr,
		&pc.RegisteredAt,
		&pc.OwnerUserID,
		&lastSeenAt,
		&pc.CreatedAt,
		&pc.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Convert connection status string to enum
	pc.ConnectionStatus = clientpc.PCConnectionStatus(connectionStatusStr)

	// Handle nullable LastSeenAt
	if lastSeenAt.Valid {
		pc.LastSeenAt = &lastSeenAt.Time
	} else {
		pc.LastSeenAt = nil
	}

	return &pc, nil
}

// scanClientPCs scans multiple rows into ClientPC structs
func (r *MySQLClientPCRepository) scanClientPCs(rows *sql.Rows) ([]*clientpc.ClientPC, error) {
	var pcs []*clientpc.ClientPC

	for rows.Next() {
		var pc clientpc.ClientPC
		var connectionStatusStr string
		var lastSeenAt sql.NullTime

		err := rows.Scan(
			&pc.PCID,
			&pc.Identifier,
			&pc.IP,
			&connectionStatusStr,
			&pc.RegisteredAt,
			&pc.OwnerUserID,
			&lastSeenAt,
			&pc.CreatedAt,
			&pc.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Convert connection status string to enum
		pc.ConnectionStatus = clientpc.PCConnectionStatus(connectionStatusStr)

		// Handle nullable LastSeenAt
		if lastSeenAt.Valid {
			pc.LastSeenAt = &lastSeenAt.Time
		} else {
			pc.LastSeenAt = nil
		}

		pcs = append(pcs, &pc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pcs, nil
}
