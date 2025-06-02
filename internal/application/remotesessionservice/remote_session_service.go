package remotesessionservice

import (
	"context"
	"fmt"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/remotesession"
	events "github.com/unikyri/escritorio-remoto-backend/internal/domain/events"
)

// RemoteSessionService servicio de aplicación para sesiones remotas
type RemoteSessionService struct {
	sessionRepo interfaces.IRemoteSessionRepository
	userRepo    interfaces.IUserRepository
	pcRepo      interfaces.IClientPCRepository
	eventBus    events.IEventBus
}

// NewRemoteSessionService crea una nueva instancia del servicio
func NewRemoteSessionService(
	sessionRepo interfaces.IRemoteSessionRepository,
	userRepo interfaces.IUserRepository,
	pcRepo interfaces.IClientPCRepository,
	eventBus events.IEventBus,
) *RemoteSessionService {
	return &RemoteSessionService{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		pcRepo:      pcRepo,
		eventBus:    eventBus,
	}
}

// InitiateSession inicia una nueva sesión de control remoto
func (rss *RemoteSessionService) InitiateSession(adminUserID, clientPCID string) (*remotesession.RemoteSession, error) {
	// Validar que el usuario administrador existe
	user, err := rss.userRepo.FindByID(adminUserID)
	if err != nil {
		return nil, fmt.Errorf("error finding admin user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("admin user not found")
	}

	// Validar que el PC cliente existe y está online
	pc, err := rss.pcRepo.FindByID(clientPCID)
	if err != nil {
		return nil, fmt.Errorf("error finding client PC: %w", err)
	}
	if pc == nil {
		return nil, fmt.Errorf("client PC not found")
	}

	// Verificar que el PC está online
	if pc.ConnectionStatus() != "ONLINE" {
		return nil, fmt.Errorf("client PC is not online")
	}

	// Crear nueva sesión
	session := remotesession.NewRemoteSession(adminUserID, clientPCID)

	// Guardar en repositorio
	err = rss.sessionRepo.Save(session)
	if err != nil {
		return nil, fmt.Errorf("error saving session: %w", err)
	}

	// Publicar evento de dominio
	event := events.NewRemoteSessionInitiatedEvent(
		session.SessionID(),
		session.AdminUserID(),
		session.ClientPCID(),
	)
	rss.eventBus.Publish(event)

	return session, nil
}

// AcceptSession acepta una sesión de control remoto
func (rss *RemoteSessionService) AcceptSession(sessionID string) error {
	// Obtener sesión
	session, err := rss.sessionRepo.FindByID(sessionID)
	if err != nil {
		return fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Aceptar sesión
	err = session.Accept()
	if err != nil {
		return fmt.Errorf("error accepting session: %w", err)
	}

	// Actualizar en repositorio
	err = rss.sessionRepo.UpdateStatus(sessionID, session.Status())
	if err != nil {
		return fmt.Errorf("error updating session status: %w", err)
	}

	// Publicar evento de dominio
	event := events.NewRemoteSessionAcceptedEvent(
		session.SessionID(),
		session.AdminUserID(),
		session.ClientPCID(),
	)
	rss.eventBus.Publish(event)

	return nil
}

// RejectSession rechaza una sesión de control remoto
func (rss *RemoteSessionService) RejectSession(sessionID, reason string) error {
	// Obtener sesión
	session, err := rss.sessionRepo.FindByID(sessionID)
	if err != nil {
		return fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Rechazar sesión
	err = session.Reject()
	if err != nil {
		return fmt.Errorf("error rejecting session: %w", err)
	}

	// Actualizar en repositorio
	err = rss.sessionRepo.UpdateStatus(sessionID, session.Status())
	if err != nil {
		return fmt.Errorf("error updating session status: %w", err)
	}

	// Publicar evento de dominio
	event := events.NewRemoteSessionRejectedEvent(
		session.SessionID(),
		session.AdminUserID(),
		session.ClientPCID(),
		reason,
	)
	rss.eventBus.Publish(event)

	return nil
}

// GetSessionById obtiene una sesión por ID
func (rss *RemoteSessionService) GetSessionById(sessionID string) (*remotesession.RemoteSession, error) {
	return rss.sessionRepo.FindByID(sessionID)
}

// GetActiveSessions obtiene todas las sesiones activas
func (rss *RemoteSessionService) GetActiveSessions() ([]*remotesession.RemoteSession, error) {
	return rss.sessionRepo.FindByStatus(remotesession.StatusActive)
}

// GetSessionsByUser obtiene las sesiones de un usuario específico
func (rss *RemoteSessionService) GetSessionsByUser(userID string) ([]*remotesession.RemoteSession, error) {
	return rss.sessionRepo.FindByAdminUserID(userID)
}

// GetSessionsByPC obtiene sesiones por PC cliente
func (rss *RemoteSessionService) GetSessionsByPC(clientPCID string) ([]*remotesession.RemoteSession, error) {
	sessions, err := rss.sessionRepo.FindByClientPCID(clientPCID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for PC: %w", err)
	}
	return sessions, nil
} 