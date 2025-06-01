package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/entities"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/repositories"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/valueobjects"
)

// ClientPCRepositoryImpl implementa IClientPCRepository usando MySQL
type ClientPCRepositoryImpl struct {
	db *sql.DB
}

// NewClientPCRepository crea una nueva instancia del repositorio
func NewClientPCRepository(db *sql.DB) repositories.IClientPCRepository {
	return &ClientPCRepositoryImpl{
		db: db,
	}
}

// Save guarda o actualiza un ClientPC
func (r *ClientPCRepositoryImpl) Save(ctx context.Context, pc *entities.ClientPC) error {
	query := `
		INSERT INTO client_pcs (
			pc_id, ip, connection_status, registered_at, owner_user_id, 
			last_seen_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			ip = VALUES(ip),
			connection_status = VALUES(connection_status),
			last_seen_at = VALUES(last_seen_at),
			updated_at = VALUES(updated_at)
	`

	_, err := r.db.ExecContext(ctx, query,
		pc.ID().Value(),
		pc.IP(),
		pc.ConnectionStatus().Value(),
		pc.RegisteredAt(),
		pc.OwnerUserID(),
		pc.LastSeenAt(),
		pc.CreatedAt(),
		pc.UpdatedAt(),
	)

	return err
}

// FindByID busca un ClientPC por su ID
func (r *ClientPCRepositoryImpl) FindByID(ctx context.Context, pcID *valueobjects.PCID) (*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE pc_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, pcID.Value())

	var pcIDStr, ip, connectionStatusStr, ownerUserID string
	var registeredAt, createdAt, updatedAt time.Time
	var lastSeenAt *time.Time

	err := row.Scan(
		&pcIDStr, &ip, &connectionStatusStr, &registeredAt,
		&ownerUserID, &lastSeenAt, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return entities.ReconstructClientPCFromDB(
		pcIDStr, "", ownerUserID, connectionStatusStr, ip,
		registeredAt, createdAt, updatedAt, lastSeenAt,
	)
}

// FindByIdentifierAndOwner busca un PC por identificador y propietario
func (r *ClientPCRepositoryImpl) FindByIdentifierAndOwner(ctx context.Context, identifier, ownerUserID string) (*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE owner_user_id = ?
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, ownerUserID)

	var pcIDStr, ip, connectionStatusStr, ownerID string
	var registeredAt, createdAt, updatedAt time.Time
	var lastSeenAt *time.Time

	err := row.Scan(
		&pcIDStr, &ip, &connectionStatusStr, &registeredAt,
		&ownerID, &lastSeenAt, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return entities.ReconstructClientPCFromDB(
		pcIDStr, identifier, ownerID, connectionStatusStr, ip,
		registeredAt, createdAt, updatedAt, lastSeenAt,
	)
}

// FindByOwner busca todos los PCs de un propietario
func (r *ClientPCRepositoryImpl) FindByOwner(ctx context.Context, ownerUserID string) ([]*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
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
func (r *ClientPCRepositoryImpl) FindOnlineByOwner(ctx context.Context, ownerUserID string) ([]*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
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
func (r *ClientPCRepositoryImpl) FindAll(ctx context.Context, limit, offset int) ([]*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
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

// FindOnlineClientPCs busca todos los PCs que están online
func (r *ClientPCRepositoryImpl) FindOnlineClientPCs(ctx context.Context) ([]*entities.ClientPC, error) {
	query := `
		SELECT pc_id, ip, connection_status, registered_at, owner_user_id,
			   last_seen_at, created_at, updated_at
		FROM client_pcs 
		WHERE connection_status = 'ONLINE'
		ORDER BY last_seen_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanClientPCs(rows)
}

// UpdateConnectionStatus actualiza solo el estado de conexión
func (r *ClientPCRepositoryImpl) UpdateConnectionStatus(ctx context.Context, pcID *valueobjects.PCID, status *valueobjects.ConnectionStatus) error {
	query := `
		UPDATE client_pcs 
		SET connection_status = ?, updated_at = ?
		WHERE pc_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, status.Value(), time.Now().UTC(), pcID.Value())
	return err
}

// UpdateLastSeen actualiza el timestamp de última conexión
func (r *ClientPCRepositoryImpl) UpdateLastSeen(ctx context.Context, pcID *valueobjects.PCID) error {
	query := `
		UPDATE client_pcs 
		SET last_seen_at = ?, updated_at = ?
		WHERE pc_id = ?
	`

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, now, now, pcID.Value())
	return err
}

// Delete elimina un ClientPC
func (r *ClientPCRepositoryImpl) Delete(ctx context.Context, pcID *valueobjects.PCID) error {
	query := `DELETE FROM client_pcs WHERE pc_id = ?`
	_, err := r.db.ExecContext(ctx, query, pcID.Value())
	return err
}

// Count retorna el número total de ClientPCs
func (r *ClientPCRepositoryImpl) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM client_pcs`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// CountOnline retorna el número de ClientPCs online
func (r *ClientPCRepositoryImpl) CountOnline(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM client_pcs WHERE connection_status = 'ONLINE'`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// scanClientPCs es un helper para escanear múltiples filas
func (r *ClientPCRepositoryImpl) scanClientPCs(rows *sql.Rows) ([]*entities.ClientPC, error) {
	var pcs []*entities.ClientPC

	for rows.Next() {
		var pcIDStr, ip, connectionStatusStr, ownerUserID string
		var registeredAt, createdAt, updatedAt time.Time
		var lastSeenAt *time.Time

		err := rows.Scan(
			&pcIDStr, &ip, &connectionStatusStr, &registeredAt,
			&ownerUserID, &lastSeenAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		pc, err := entities.ReconstructClientPCFromDB(
			pcIDStr, "", ownerUserID, connectionStatusStr, ip,
			registeredAt, createdAt, updatedAt, lastSeenAt,
		)
		if err != nil {
			return nil, err
		}

		pcs = append(pcs, pc)
	}

	return pcs, rows.Err()
}
