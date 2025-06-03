package actionlogservice

import (
	"context"
	"fmt"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/actionlog"
)

// IActionLogService define la interfaz para operaciones de negocio de ActionLog
type IActionLogService interface {
	// LogAction registra una nueva acción en el sistema
	LogAction(ctx context.Context, actionType actionlog.ActionType, description string, 
		performedByUserID string, subjectEntityID *string, subjectEntityType *string, 
		details map[string]interface{}) error

	// LogSessionEnded registra cuando una sesión es finalizada
	LogSessionEnded(ctx context.Context, sessionID, adminUserID, reason string) error

	// GetRecentLogs obtiene los logs más recientes
	GetRecentLogs(ctx context.Context, limit int) ([]*actionlog.ActionLog, error)

	// GetLogsByUser obtiene los logs de un usuario específico
	GetLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*actionlog.ActionLog, error)

	// GetLogsByActionType obtiene logs por tipo de acción
	GetLogsByActionType(ctx context.Context, actionType actionlog.ActionType, limit, offset int) ([]*actionlog.ActionLog, error)

	// GetLogsByEntity obtiene logs relacionados con una entidad específica
	GetLogsByEntity(ctx context.Context, entityID, entityType string, limit, offset int) ([]*actionlog.ActionLog, error)

	// GetLogsCount obtiene el número total de logs
	GetLogsCount(ctx context.Context) (int, error)
}

// ActionLogService implementa la lógica de negocio para ActionLog
type ActionLogService struct {
	actionLogRepo interfaces.IActionLogRepository
}

// NewActionLogService crea una nueva instancia del servicio
func NewActionLogService(actionLogRepo interfaces.IActionLogRepository) IActionLogService {
	return &ActionLogService{
		actionLogRepo: actionLogRepo,
	}
}

// LogAction registra una nueva acción en el sistema
func (als *ActionLogService) LogAction(ctx context.Context, actionType actionlog.ActionType, 
	description string, performedByUserID string, subjectEntityID *string, 
	subjectEntityType *string, details map[string]interface{}) error {

	// Crear nueva entrada de log
	log := actionlog.NewActionLog(
		actionType,
		description,
		performedByUserID,
		subjectEntityID,
		subjectEntityType,
		details,
	)

	// Guardar en repositorio
	err := als.actionLogRepo.Save(ctx, log)
	if err != nil {
		return fmt.Errorf("error saving action log: %w", err)
	}

	return nil
}

// LogSessionEnded registra cuando una sesión es finalizada
func (als *ActionLogService) LogSessionEnded(ctx context.Context, sessionID, adminUserID, reason string) error {
	description := fmt.Sprintf("Sesión de control remoto finalizada por administrador - Razón: %s", reason)
	
	details := map[string]interface{}{
		"session_id": sessionID,
		"reason":     reason,
		"ended_by":   "admin",
	}

	subjectEntityID := sessionID
	subjectEntityType := "REMOTE_SESSION"

	return als.LogAction(
		ctx,
		actionlog.ActionRemoteSessionEnded,
		description,
		adminUserID,
		&subjectEntityID,
		&subjectEntityType,
		details,
	)
}

// GetRecentLogs obtiene los logs más recientes
func (als *ActionLogService) GetRecentLogs(ctx context.Context, limit int) ([]*actionlog.ActionLog, error) {
	logs, err := als.actionLogRepo.FindRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("error getting recent logs: %w", err)
	}
	return logs, nil
}

// GetLogsByUser obtiene los logs de un usuario específico
func (als *ActionLogService) GetLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*actionlog.ActionLog, error) {
	logs, err := als.actionLogRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error getting logs by user: %w", err)
	}
	return logs, nil
}

// GetLogsByActionType obtiene logs por tipo de acción
func (als *ActionLogService) GetLogsByActionType(ctx context.Context, actionType actionlog.ActionType, limit, offset int) ([]*actionlog.ActionLog, error) {
	logs, err := als.actionLogRepo.FindByActionType(ctx, actionType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error getting logs by action type: %w", err)
	}
	return logs, nil
}

// GetLogsByEntity obtiene logs relacionados con una entidad específica
func (als *ActionLogService) GetLogsByEntity(ctx context.Context, entityID, entityType string, limit, offset int) ([]*actionlog.ActionLog, error) {
	logs, err := als.actionLogRepo.FindBySubjectEntity(ctx, entityID, entityType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error getting logs by entity: %w", err)
	}
	return logs, nil
}

// GetLogsCount obtiene el número total de logs
func (als *ActionLogService) GetLogsCount(ctx context.Context) (int, error) {
	count, err := als.actionLogRepo.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting logs count: %w", err)
	}
	return count, nil
} 