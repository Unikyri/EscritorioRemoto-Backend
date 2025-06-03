package interfaces

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/actionlog"
)

// IActionLogRepository define la interfaz para operaciones de persistencia de ActionLog
type IActionLogRepository interface {
	// Save guarda una nueva entrada de log de auditoría
	Save(ctx context.Context, log *actionlog.ActionLog) error

	// FindByID busca un log por su ID
	FindByID(ctx context.Context, logID int64) (*actionlog.ActionLog, error)

	// FindByUserID busca logs por ID de usuario que los ejecutó
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*actionlog.ActionLog, error)

	// FindByActionType busca logs por tipo de acción
	FindByActionType(ctx context.Context, actionType actionlog.ActionType, limit, offset int) ([]*actionlog.ActionLog, error)

	// FindBySubjectEntity busca logs por entidad objetivo
	FindBySubjectEntity(ctx context.Context, entityID, entityType string, limit, offset int) ([]*actionlog.ActionLog, error)

	// FindRecent busca los logs más recientes
	FindRecent(ctx context.Context, limit int) ([]*actionlog.ActionLog, error)

	// Count retorna el número total de logs
	Count(ctx context.Context) (int, error)

	// CountByUser retorna el número de logs de un usuario específico
	CountByUser(ctx context.Context, userID string) (int, error)
}
