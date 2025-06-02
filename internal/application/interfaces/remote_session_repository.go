package interfaces

import (
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/remotesession"
)

// IRemoteSessionRepository define la interface del repositorio de sesiones remotas
type IRemoteSessionRepository interface {
	// Save guarda una nueva sesión remota
	Save(session *remotesession.RemoteSession) error
	
	// FindById busca una sesión por su ID
	FindById(id string) (*remotesession.RemoteSession, error)
	
	// UpdateStatus actualiza el estado de una sesión
	UpdateStatus(id string, status remotesession.SessionStatus) error
	
	// FindByAdminUserID busca sesiones por ID de usuario administrador
	FindByAdminUserID(adminUserID string) ([]*remotesession.RemoteSession, error)
	
	// FindByClientPCID busca sesiones por ID de PC cliente
	FindByClientPCID(clientPCID string) ([]*remotesession.RemoteSession, error)
	
	// FindActiveSessions busca sesiones activas
	FindActiveSessions() ([]*remotesession.RemoteSession, error)
	
	// FindPendingSessions busca sesiones pendientes de aprobación
	FindPendingSessions() ([]*remotesession.RemoteSession, error)
	
	// Update actualiza una sesión completa
	Update(session *remotesession.RemoteSession) error
	
	// Delete elimina una sesión (soft delete)
	Delete(id string) error
	
	// FindByStatus busca sesiones por estado
	FindByStatus(status remotesession.SessionStatus) ([]*remotesession.RemoteSession, error)
	
	// FindSessionsByDateRange busca sesiones en un rango de fechas
	FindSessionsByDateRange(adminUserID string, startDate, endDate string) ([]*remotesession.RemoteSession, error)
	
	// CountSessionsByUser cuenta sesiones por usuario
	CountSessionsByUser(adminUserID string) (int64, error)
} 