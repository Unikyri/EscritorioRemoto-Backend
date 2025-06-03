package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// ClientPCRepositoryImpl implementa el repositorio de ClientPC usando MySQL
type ClientPCRepositoryImpl struct {
	db *sql.DB
}

// NewClientPCRepository crea una nueva instancia del repositorio
func NewClientPCRepository(db *sql.DB) interfaces.IClientPCRepository {
	return &ClientPCRepositoryImpl{
		db: db,
	}
}

// Save guarda o actualiza un ClientPC
func (r *ClientPCRepositoryImpl) Save(ctx context.Context, pc *clientpc.ClientPC) error {
	query := `
		INSERT INTO client_pcs (
			pc_id, identifier, ip, connection_status, registered_at, owner_user_id, 
			last_seen_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			identifier = VALUES(identifier),
			ip = VALUES(ip),
			connection_status = VALUES(connection_status),
			last_seen_at = VALUES(last_seen_at),
			updated_at = VALUES(updated_at)
	`

	_, err := r.db.ExecContext(ctx, query,
		pc.PCID,
		pc.Identifier,
		pc.IP,
		string(pc.ConnectionStatus),
		pc.RegisteredAt,
		pc.OwnerUserID,
		pc.LastSeenAt,
		pc.CreatedAt,
		pc.UpdatedAt,
	)

	return err
}

// FindByID busca un ClientPC por su ID
func (r *ClientPCRepositoryImpl) FindByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE pc_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, pcID)

	var pcIDStr, identifier, ip, connectionStatusStr, ownerUserID string
	var registeredAt, createdAt, updatedAt time.Time
	var lastSeenAt *time.Time

	err := row.Scan(
		&pcIDStr, &identifier, &ip, &connectionStatusStr, &registeredAt,
		&ownerUserID, &lastSeenAt, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Construir la entidad simple
	pc := &clientpc.ClientPC{
		PCID:             pcIDStr,
		Identifier:       identifier,
		IP:               ip,
		ConnectionStatus: clientpc.PCConnectionStatus(connectionStatusStr),
		RegisteredAt:     registeredAt,
		OwnerUserID:      ownerUserID,
		LastSeenAt:       lastSeenAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	return pc, nil
}

// FindByIdentifierAndOwner busca un PC por identificador y propietario
func (r *ClientPCRepositoryImpl) FindByIdentifierAndOwner(ctx context.Context, identifier, ownerUserID string) (*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE identifier = ? AND owner_user_id = ?
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, identifier, ownerUserID)

	var pcIDStr, identifierCol, ip, connectionStatusStr, ownerID string
	var registeredAt, createdAt, updatedAt time.Time
	var lastSeenAt *time.Time

	err := row.Scan(
		&pcIDStr, &identifierCol, &ip, &connectionStatusStr, &registeredAt,
		&ownerID, &lastSeenAt, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Construir la entidad simple
	pc := &clientpc.ClientPC{
		PCID:             pcIDStr,
		Identifier:       identifierCol,
		IP:               ip,
		ConnectionStatus: clientpc.PCConnectionStatus(connectionStatusStr),
		RegisteredAt:     registeredAt,
		OwnerUserID:      ownerID,
		LastSeenAt:       lastSeenAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	return pc, nil
}

// FindByOwner busca todos los PCs de un propietario
func (r *ClientPCRepositoryImpl) FindByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE owner_user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// FindOnlineByOwner busca todos los PCs online de un propietario
func (r *ClientPCRepositoryImpl) FindOnlineByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE owner_user_id = ? AND connection_status = 'ONLINE'
		ORDER BY last_seen_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// FindAll busca todos los ClientPCs con paginación
func (r *ClientPCRepositoryImpl) FindAll(ctx context.Context, limit, offset int) ([]*clientpc.ClientPC, error) {
	query := `
		SELECT pc_id, identifier, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// UpdateConnectionStatus actualiza solo el estado de conexión
func (r *ClientPCRepositoryImpl) UpdateConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error {
	query := `
		UPDATE client_pcs 
		SET connection_status = ?, updated_at = ?
		WHERE pc_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, string(status), time.Now().UTC(), pcID)
	return err
}

// UpdateLastSeen actualiza el timestamp de última conexión
func (r *ClientPCRepositoryImpl) UpdateLastSeen(ctx context.Context, pcID string) error {
	query := `
		UPDATE client_pcs 
		SET last_seen_at = ?, updated_at = ?
		WHERE pc_id = ?
	`

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, now, now, pcID)
	return err
}

// Delete elimina un ClientPC
func (r *ClientPCRepositoryImpl) Delete(ctx context.Context, pcID string) error {
	query := `DELETE FROM client_pcs WHERE pc_id = ?`
	_, err := r.db.ExecContext(ctx, query, pcID)
	return err
}

// CountByOwner retorna el número de PCs de un propietario específico
func (r *ClientPCRepositoryImpl) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	query := `SELECT COUNT(*) FROM client_pcs WHERE owner_user_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, ownerID).Scan(&count)
	return count, err
}

// scanClientPCs es un helper para escanear múltiples filas
func (r *ClientPCRepositoryImpl) scanClientPCs(rows *sql.Rows) ([]*clientpc.ClientPC, error) {
	var pcs []*clientpc.ClientPC

	for rows.Next() {
		var pcIDStr, identifier, ip, connectionStatusStr, ownerUserID string
		var registeredAt, createdAt, updatedAt time.Time
		var lastSeenAt *time.Time

		err := rows.Scan(
			&pcIDStr, &identifier, &ip, &connectionStatusStr, &registeredAt,
			&ownerUserID, &lastSeenAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Construir la entidad simple
		pc := &clientpc.ClientPC{
			PCID:             pcIDStr,
			Identifier:       identifier,
			IP:               ip,
			ConnectionStatus: clientpc.PCConnectionStatus(connectionStatusStr),
			RegisteredAt:     registeredAt,
			OwnerUserID:      ownerUserID,
			LastSeenAt:       lastSeenAt,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}

		pcs = append(pcs, pc)
	}

	return pcs, rows.Err()
}
