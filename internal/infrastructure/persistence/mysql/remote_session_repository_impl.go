package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/remotesession"
)

// RemoteSessionRepositoryImpl implementa IRemoteSessionRepository usando MySQL
type RemoteSessionRepositoryImpl struct {
	db *sql.DB
}

// NewRemoteSessionRepository crea una nueva instancia del repositorio
func NewRemoteSessionRepository(db *sql.DB) interfaces.IRemoteSessionRepository {
	return &RemoteSessionRepositoryImpl{
		db: db,
	}
}

// Save guarda una nueva sesión remota
func (rsr *RemoteSessionRepositoryImpl) Save(session *remotesession.RemoteSession) error {
	query := `
		INSERT INTO remote_sessions (
			session_id, admin_user_id, client_pc_id, start_time, end_time, 
			status, session_video_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := rsr.db.Exec(
		query,
		session.SessionID(),
		session.AdminUserID(),
		session.ClientPCID(),
		session.StartTime(),
		session.EndTime(),
		string(session.Status()),
		session.SessionVideoID(),
		session.CreatedAt(),
		session.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to save remote session: %w", err)
	}

	return nil
}

// FindById busca una sesión por su ID
func (rsr *RemoteSessionRepositoryImpl) FindById(id string) (*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE session_id = ?
	`

	row := rsr.db.QueryRow(query, id)

	var sessionID, adminUserID, clientPCID, status string
	var sessionVideoID sql.NullString
	var startTime, endTime sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&sessionID, &adminUserID, &clientPCID,
		&startTime, &endTime, &status, &sessionVideoID,
		&createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find remote session: %w", err)
	}

	// Reconstruir la entidad desde la base de datos
	session := rsr.reconstructSession(
		sessionID, adminUserID, clientPCID,
		startTime, endTime, status, sessionVideoID,
		createdAt, updatedAt,
	)

	return session, nil
}

// UpdateStatus actualiza el estado de una sesión
func (rsr *RemoteSessionRepositoryImpl) UpdateStatus(id string, status remotesession.SessionStatus) error {
	query := `
		UPDATE remote_sessions 
		SET status = ?, updated_at = ?
		WHERE session_id = ?
	`

	result, err := rsr.db.Exec(query, string(status), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session with ID %s not found", id)
	}

	return nil
}

// Update actualiza una sesión completa
func (rsr *RemoteSessionRepositoryImpl) Update(session *remotesession.RemoteSession) error {
	query := `
		UPDATE remote_sessions 
		SET admin_user_id = ?, client_pc_id = ?, start_time = ?, end_time = ?,
			status = ?, session_video_id = ?, updated_at = ?
		WHERE session_id = ?
	`

	result, err := rsr.db.Exec(
		query,
		session.AdminUserID(),
		session.ClientPCID(),
		session.StartTime(),
		session.EndTime(),
		string(session.Status()),
		session.SessionVideoID(),
		session.UpdatedAt(),
		session.SessionID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update remote session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session with ID %s not found", session.SessionID())
	}

	return nil
}

// FindByAdminUserID busca sesiones por ID de usuario administrador
func (rsr *RemoteSessionRepositoryImpl) FindByAdminUserID(adminUserID string) ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE admin_user_id = ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, adminUserID)
}

// FindByClientPCID busca sesiones por ID de PC cliente
func (rsr *RemoteSessionRepositoryImpl) FindByClientPCID(clientPCID string) ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE client_pc_id = ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, clientPCID)
}

// FindActiveSessions busca sesiones activas
func (rsr *RemoteSessionRepositoryImpl) FindActiveSessions() ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE status = ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, string(remotesession.StatusActive))
}

// FindPendingSessions busca sesiones pendientes de aprobación
func (rsr *RemoteSessionRepositoryImpl) FindPendingSessions() ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE status = ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, string(remotesession.StatusPendingApproval))
}

// FindByStatus busca sesiones por estado
func (rsr *RemoteSessionRepositoryImpl) FindByStatus(status remotesession.SessionStatus) ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE status = ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, string(status))
}

// FindSessionsByDateRange busca sesiones en un rango de fechas
func (rsr *RemoteSessionRepositoryImpl) FindSessionsByDateRange(adminUserID string, startDate, endDate string) ([]*remotesession.RemoteSession, error) {
	query := `
		SELECT session_id, admin_user_id, client_pc_id, start_time, end_time,
			   status, session_video_id, created_at, updated_at
		FROM remote_sessions
		WHERE admin_user_id = ? AND created_at BETWEEN ? AND ?
		ORDER BY created_at DESC
	`

	return rsr.findSessions(query, adminUserID, startDate, endDate)
}

// CountSessionsByUser cuenta sesiones por usuario
func (rsr *RemoteSessionRepositoryImpl) CountSessionsByUser(adminUserID string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM remote_sessions
		WHERE admin_user_id = ?
	`

	var count int64
	err := rsr.db.QueryRow(query, adminUserID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}

// Delete elimina una sesión (soft delete)
func (rsr *RemoteSessionRepositoryImpl) Delete(id string) error {
	// Implementar soft delete marcando como eliminado
	// Por ahora implementamos delete físico
	query := `DELETE FROM remote_sessions WHERE session_id = ?`

	result, err := rsr.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session with ID %s not found", id)
	}

	return nil
}

// Métodos auxiliares privados

// findSessions ejecuta una query y retorna las sesiones encontradas
func (rsr *RemoteSessionRepositoryImpl) findSessions(query string, args ...interface{}) ([]*remotesession.RemoteSession, error) {
	rows, err := rsr.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var sessions []*remotesession.RemoteSession

	for rows.Next() {
		var sessionID, adminUserID, clientPCID, status string
		var sessionVideoID sql.NullString
		var startTime, endTime sql.NullTime
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&sessionID, &adminUserID, &clientPCID,
			&startTime, &endTime, &status, &sessionVideoID,
			&createdAt, &updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session := rsr.reconstructSession(
			sessionID, adminUserID, clientPCID,
			startTime, endTime, status, sessionVideoID,
			createdAt, updatedAt,
		)

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return sessions, nil
}

// reconstructSession reconstruye una entidad RemoteSession desde datos de BD
func (rsr *RemoteSessionRepositoryImpl) reconstructSession(
	sessionID, adminUserID, clientPCID string,
	startTime, endTime sql.NullTime,
	status string,
	sessionVideoID sql.NullString,
	createdAt, updatedAt time.Time,
) *remotesession.RemoteSession {
	// Convertir sql.NullTime a *time.Time
	var startTimePtr, endTimePtr *time.Time
	if startTime.Valid {
		startTimePtr = &startTime.Time
	}
	if endTime.Valid {
		endTimePtr = &endTime.Time
	}
	
	// Convertir sql.NullString a *string
	var sessionVideoIDPtr *string
	if sessionVideoID.Valid {
		sessionVideoIDPtr = &sessionVideoID.String
	}
	
	// Usar el factory method para reconstruir la entidad
	return remotesession.NewRemoteSessionFromDB(
		sessionID,
		adminUserID,
		clientPCID,
		startTimePtr,
		endTimePtr,
		remotesession.SessionStatus(status),
		sessionVideoIDPtr,
		createdAt,
		updatedAt,
	)
} 