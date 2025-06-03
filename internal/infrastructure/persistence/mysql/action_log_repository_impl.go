package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/actionlog"
)

// ActionLogRepositoryImpl implementa el repositorio de ActionLog usando MySQL
type ActionLogRepositoryImpl struct {
	db *sql.DB
}

// NewActionLogRepository crea una nueva instancia del repositorio
func NewActionLogRepository(db *sql.DB) interfaces.IActionLogRepository {
	return &ActionLogRepositoryImpl{
		db: db,
	}
}

// Save guarda una nueva entrada de log de auditoría
func (r *ActionLogRepositoryImpl) Save(ctx context.Context, log *actionlog.ActionLog) error {
	// Convertir details a JSON
	var detailsJSON []byte
	var err error

	if log.Details() != nil {
		detailsJSON, err = json.Marshal(log.Details())
		if err != nil {
			return fmt.Errorf("error marshalling details to JSON: %w", err)
		}
	}

	query := `
		INSERT INTO action_logs (
			timestamp, action_type, description, performed_by_user_id, 
			subject_entity_id, subject_entity_type, details, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		log.Timestamp(),
		string(log.ActionType()),
		log.Description(),
		log.PerformedByUserID(),
		log.SubjectEntityID(),
		log.SubjectEntityType(),
		detailsJSON,
		log.CreatedAt(),
	)

	if err != nil {
		return fmt.Errorf("error saving action log: %w", err)
	}

	// Obtener el ID asignado y establecerlo en la entidad
	logID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert ID: %w", err)
	}

	log.SetLogID(logID)
	return nil
}

// FindByID busca un log por su ID
func (r *ActionLogRepositoryImpl) FindByID(ctx context.Context, logID int64) (*actionlog.ActionLog, error) {
	query := `
		SELECT log_id, timestamp, action_type, description, performed_by_user_id, 
		       subject_entity_id, subject_entity_type, details, created_at
		FROM action_logs 
		WHERE log_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, logID)
	return r.scanActionLog(row)
}

// FindByUserID busca logs por ID de usuario que los ejecutó
func (r *ActionLogRepositoryImpl) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*actionlog.ActionLog, error) {
	query := `
		SELECT log_id, timestamp, action_type, description, performed_by_user_id, 
		       subject_entity_id, subject_entity_type, details, created_at
		FROM action_logs 
		WHERE performed_by_user_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error finding action logs by user ID: %w", err)
	}
	defer rows.Close()

	return r.scanActionLogs(rows)
}

// FindByActionType busca logs por tipo de acción
func (r *ActionLogRepositoryImpl) FindByActionType(ctx context.Context, actionType actionlog.ActionType, limit, offset int) ([]*actionlog.ActionLog, error) {
	query := `
		SELECT log_id, timestamp, action_type, description, performed_by_user_id, 
		       subject_entity_id, subject_entity_type, details, created_at
		FROM action_logs 
		WHERE action_type = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, string(actionType), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error finding action logs by type: %w", err)
	}
	defer rows.Close()

	return r.scanActionLogs(rows)
}

// FindBySubjectEntity busca logs por entidad objetivo
func (r *ActionLogRepositoryImpl) FindBySubjectEntity(ctx context.Context, entityID, entityType string, limit, offset int) ([]*actionlog.ActionLog, error) {
	query := `
		SELECT log_id, timestamp, action_type, description, performed_by_user_id, 
		       subject_entity_id, subject_entity_type, details, created_at
		FROM action_logs 
		WHERE subject_entity_id = ? AND subject_entity_type = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, entityID, entityType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error finding action logs by subject entity: %w", err)
	}
	defer rows.Close()

	return r.scanActionLogs(rows)
}

// FindRecent busca los logs más recientes
func (r *ActionLogRepositoryImpl) FindRecent(ctx context.Context, limit int) ([]*actionlog.ActionLog, error) {
	query := `
		SELECT log_id, timestamp, action_type, description, performed_by_user_id, 
		       subject_entity_id, subject_entity_type, details, created_at
		FROM action_logs 
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("error finding recent action logs: %w", err)
	}
	defer rows.Close()

	return r.scanActionLogs(rows)
}

// Count retorna el número total de logs
func (r *ActionLogRepositoryImpl) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM action_logs`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting action logs: %w", err)
	}

	return count, nil
}

// CountByUser retorna el número de logs de un usuario específico
func (r *ActionLogRepositoryImpl) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM action_logs WHERE performed_by_user_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting action logs by user: %w", err)
	}

	return count, nil
}

// Helper methods

// scanActionLog escanea una fila en un ActionLog
func (r *ActionLogRepositoryImpl) scanActionLog(row *sql.Row) (*actionlog.ActionLog, error) {
	var actionTypeStr string
	var subjectEntityID, subjectEntityType sql.NullString
	var detailsJSON sql.NullString
	var logID int64
	var timestamp, createdAt time.Time
	var description, performedByUserID string

	err := row.Scan(
		&logID,
		&timestamp,
		&actionTypeStr,
		&description,
		&performedByUserID,
		&subjectEntityID,
		&subjectEntityType,
		&detailsJSON,
		&createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error scanning action log: %w", err)
	}

	// Convertir campos nullable
	var entityID, entityType *string
	if subjectEntityID.Valid {
		entityID = &subjectEntityID.String
	}
	if subjectEntityType.Valid {
		entityType = &subjectEntityType.String
	}

	// Convertir JSON details
	var details map[string]interface{}
	if detailsJSON.Valid && detailsJSON.String != "" {
		err = json.Unmarshal([]byte(detailsJSON.String), &details)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling details JSON: %w", err)
		}
	}

	// Crear ActionLog desde DB
	actionLog := actionlog.NewActionLogFromDB(
		logID,
		timestamp,
		actionlog.ActionType(actionTypeStr),
		description,
		performedByUserID,
		entityID,
		entityType,
		details,
		createdAt,
	)

	return actionLog, nil
}

// scanActionLogs escanea múltiples filas en ActionLogs
func (r *ActionLogRepositoryImpl) scanActionLogs(rows *sql.Rows) ([]*actionlog.ActionLog, error) {
	var logs []*actionlog.ActionLog

	for rows.Next() {
		var actionTypeStr string
		var subjectEntityID, subjectEntityType sql.NullString
		var detailsJSON sql.NullString
		var logID int64
		var timestamp, createdAt time.Time
		var description, performedByUserID string

		err := rows.Scan(
			&logID,
			&timestamp,
			&actionTypeStr,
			&description,
			&performedByUserID,
			&subjectEntityID,
			&subjectEntityType,
			&detailsJSON,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning action log row: %w", err)
		}

		// Convertir campos nullable
		var entityID, entityType *string
		if subjectEntityID.Valid {
			entityID = &subjectEntityID.String
		}
		if subjectEntityType.Valid {
			entityType = &subjectEntityType.String
		}

		// Convertir JSON details
		var details map[string]interface{}
		if detailsJSON.Valid && detailsJSON.String != "" {
			err = json.Unmarshal([]byte(detailsJSON.String), &details)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling details JSON: %w", err)
			}
		}

		// Crear ActionLog desde DB
		actionLog := actionlog.NewActionLogFromDB(
			logID,
			timestamp,
			actionlog.ActionType(actionTypeStr),
			description,
			performedByUserID,
			entityID,
			entityType,
			details,
			createdAt,
		)

		logs = append(logs, actionLog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating action log rows: %w", err)
	}

	return logs, nil
}
