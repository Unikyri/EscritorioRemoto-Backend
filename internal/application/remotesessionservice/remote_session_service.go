package remotesessionservice

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	events "github.com/unikyri/escritorio-remoto-backend/internal/domain/events"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/remotesession"
)

// RemoteSessionService servicio de aplicación para sesiones remotas
type RemoteSessionService struct {
	sessionRepo interfaces.IRemoteSessionRepository
	userRepo    interfaces.IUserRepository
	pcRepo      interfaces.IClientPCRepository
	eventBus    events.IEventBus

	// Callback para notificar al AdminWebSocketHandler
	notifySessionEndedCallback func(sessionID, clientPCID, adminUserID string)
	// Callback para notificar al cliente cuando termina la sesión
	notifyClientSessionEndedCallback func(sessionID, clientPCID string)
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

// SetSessionEndedNotifier establece el callback para notificar cuando una sesión termina
func (rss *RemoteSessionService) SetSessionEndedNotifier(callback func(sessionID, clientPCID, adminUserID string)) {
	rss.notifySessionEndedCallback = callback
}

// SetClientSessionEndedNotifier establece el callback para notificar al cliente cuando una sesión termina
func (rss *RemoteSessionService) SetClientSessionEndedNotifier(callback func(sessionID, clientPCID string)) {
	rss.notifyClientSessionEndedCallback = callback
}

// CleanupStuckSessions limpia sesiones que se quedaron en estado activo o pendiente sin resolución.
func (rss *RemoteSessionService) CleanupStuckSessions(clientPCID string) error {
	sessions, err := rss.sessionRepo.FindByClientPCID(clientPCID)
	if err != nil {
		return fmt.Errorf("failed to get sessions for cleanup: %w", err)
	}

	now := time.Now().UTC()

	for _, session := range sessions {
		originalStatus := session.Status()
		var actionTaken bool = false
		var newRepoStatus remotesession.SessionStatus // Para registrar el estado que se intentará guardar en el repo
		var internalError error

		if originalStatus == remotesession.StatusActive {
			stuckTimeoutActive := 15 * time.Minute
			shouldClean := false
			reason := ""
			if session.StartTime() == nil && now.Sub(session.CreatedAt()) > stuckTimeoutActive {
				shouldClean = true
				reason = fmt.Sprintf("active session %s with nil StartTime, created %v ago", session.SessionID(), now.Sub(session.CreatedAt()))
			} else if session.StartTime() != nil && now.Sub(*session.StartTime()) > stuckTimeoutActive {
				shouldClean = true
				reason = fmt.Sprintf("active session %s started %v ago", session.SessionID(), now.Sub(*session.StartTime()))
			}

			if shouldClean {
				log.Printf("🧹 Cleaning up stuck ACTIVE session: %s (%s)", session.SessionID(), reason)
				internalError = session.End(remotesession.StatusFailed)
				if internalError != nil {
					log.Printf("⚠️ Warning: Call to End() on stuck ACTIVE session %s returned error: %v. Entity status before End(): %s, after End(): %s",
						session.SessionID(), internalError, originalStatus, session.Status())
				}
				newRepoStatus = session.Status() // El estado que la entidad tenga ahora
				actionTaken = true
			}
		} else if originalStatus == remotesession.StatusPendingApproval {
			stuckTimeoutPending := 2 * time.Minute
			if now.Sub(session.CreatedAt()) > stuckTimeoutPending {
				reason := fmt.Sprintf("pending approval session %s created %v ago", session.SessionID(), now.Sub(session.CreatedAt()))
				log.Printf("🧹 Cleaning up stuck PENDING_APPROVAL session: %s (%s)", session.SessionID(), reason)

				internalError = session.Reject() // Usar Reject para PENDING_APPROVAL
				if internalError != nil {
					log.Printf("⚠️ Warning: Call to Reject() on stuck PENDING_APPROVAL session %s returned error: %v. Entity status before Reject(): %s, after Reject(): %s",
						session.SessionID(), internalError, originalStatus, session.Status())
					// Si Reject() falla, podría ser que el estado ya cambió. El estado actual de la entidad se usará para el repo.
				}
				newRepoStatus = session.Status() // El estado que la entidad tenga ahora (debería ser REJECTED o el estado original si Reject falló)
				actionTaken = true
			}
		} else if originalStatus == remotesession.StatusRejected {
			// Limpiar sesiones rechazadas antiguas para evitar acumulación
			stuckTimeoutRejected := 30 * time.Minute
			if now.Sub(session.CreatedAt()) > stuckTimeoutRejected {
				reason := fmt.Sprintf("rejected session %s created %v ago", session.SessionID(), now.Sub(session.CreatedAt()))
				log.Printf("🧹 Cleaning up old REJECTED session: %s (%s)", session.SessionID(), reason)

				// Para sesiones rechazadas, simplemente las marcamos como procesadas
				// No intentamos cambiar su estado con End() porque ya están en estado final
				log.Printf("ℹ️ REJECTED session %s marked for cleanup (no state change needed)", session.SessionID())

				// Actualizar timestamp para evitar que se procese repetidamente
				errUpdate := rss.sessionRepo.UpdateStatus(session.SessionID(), remotesession.StatusRejected)
				if errUpdate != nil {
					log.Printf("❌ Failed to update timestamp for REJECTED session %s: %v", session.SessionID(), errUpdate)
				} else {
					log.Printf("✅ REJECTED session %s cleanup timestamp updated", session.SessionID())
				}

				// No marcar como actionTaken porque no cambiamos el estado de la entidad
				continue
			}
		}

		if actionTaken {
			// Solo actualizar si el estado de la entidad realmente cambió o si hubo un intento de cambiarlo
			if newRepoStatus != originalStatus || internalError == nil { // internalError == nil significa que la operación (End/Reject) tuvo éxito en cambiar el estado o no era necesaria
				errUpdate := rss.sessionRepo.UpdateStatus(session.SessionID(), newRepoStatus)
				if errUpdate != nil {
					log.Printf("❌ CRITICAL: Failed to update session %s status in repo (original: %s, attempted new: %s): %v",
						session.SessionID(), originalStatus, newRepoStatus, errUpdate)
				} else {
					log.Printf("✅ Session %s processed. Original status: %s, New status in repo: %s",
						session.SessionID(), originalStatus, newRepoStatus)
				}
			} else {
				log.Printf("ℹ️ Session %s (status: %s) was processed by cleanup, but no state change was made or repo update needed due to internal error during state change.",
					session.SessionID(), originalStatus)
			}
		}
	}
	return nil
}

// InitiateSession inicia una nueva sesión de control remoto (método actualizado)
func (rss *RemoteSessionService) InitiateSession(adminUserID, clientPCID string) (*remotesession.RemoteSession, error) {
	// Limpiar sesiones anteriores que puedan estar stuck
	err := rss.CleanupStuckSessions(clientPCID)
	if err != nil {
		log.Printf("⚠️ Warning during cleanup: %v", err)
	}

	// Verificar que no hay una sesión activa para este PC
	activeSession, err := rss.GetActiveSessionForPC(clientPCID)
	if err != nil {
		return nil, fmt.Errorf("error checking active sessions: %w", err)
	}
	if activeSession != nil {
		return nil, fmt.Errorf("session already active: %s", activeSession.SessionID())
	}

	// Validar que el usuario administrador existe
	user, err := rss.userRepo.FindByID(adminUserID)
	if err != nil {
		return nil, fmt.Errorf("error finding admin user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("admin user not found")
	}

	// Validar que el PC cliente existe y está online
	pc, err := rss.pcRepo.FindByID(context.Background(), clientPCID)
	if err != nil {
		return nil, fmt.Errorf("error finding client PC: %w", err)
	}
	if pc == nil {
		return nil, fmt.Errorf("client PC not found")
	}

	// Verificar que el PC está online
	if string(pc.ConnectionStatus) != "ONLINE" {
		return nil, fmt.Errorf("client PC is not online")
	}

	// Crear nueva sesión
	session, err := remotesession.NewRemoteSession(adminUserID, clientPCID)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

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
		"", // adminUsername - podríamos obtenerlo del usuario si es necesario
		"", // clientPCName - podríamos obtenerlo del PC si es necesario
	)
	rss.eventBus.Publish(event)

	return session, nil
}

// AcceptSession acepta una sesión de control remoto
func (rss *RemoteSessionService) AcceptSession(sessionID string) error {
	// Obtener sesión
	session, err := rss.sessionRepo.FindById(sessionID)
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
		*session.StartTime(),
	)
	rss.eventBus.Publish(event)

	return nil
}

// RejectSession rechaza una sesión de control remoto
func (rss *RemoteSessionService) RejectSession(sessionID, reason string) error {
	// Obtener sesión
	session, err := rss.sessionRepo.FindById(sessionID)
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
	)
	rss.eventBus.Publish(event)

	return nil
}

// GetSessionById obtiene una sesión por ID
func (rss *RemoteSessionService) GetSessionById(sessionID string) (*remotesession.RemoteSession, error) {
	return rss.sessionRepo.FindById(sessionID)
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

// IsSessionActiveForStreaming verifica si una sesión está activa para streaming de pantalla
func (rss *RemoteSessionService) IsSessionActiveForStreaming(sessionID string) (bool, error) {
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return false, fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return false, fmt.Errorf("session not found")
	}

	// Solo permitir streaming si la sesión está en estado ACTIVE
	return session.Status() == remotesession.StatusActive, nil
}

// GetActiveSessionForPC obtiene la sesión activa para un PC específico (si existe)
func (rss *RemoteSessionService) GetActiveSessionForPC(clientPCID string) (*remotesession.RemoteSession, error) {
	sessions, err := rss.sessionRepo.FindByClientPCID(clientPCID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for PC: %w", err)
	}

	// Buscar sesión activa
	for _, session := range sessions {
		if session.Status() == remotesession.StatusActive {
			return session, nil
		}
	}

	return nil, nil // No hay sesión activa
}

// GetAdminUserIDForActiveSession obtiene el ID del administrador para una sesión activa
func (rss *RemoteSessionService) GetAdminUserIDForActiveSession(sessionID string) (string, error) {
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return "", fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	// Verificar que está activa
	if session.Status() != remotesession.StatusActive {
		return "", fmt.Errorf("session is not active")
	}

	return session.AdminUserID(), nil
}

// GetClientPCIDForActiveSession obtiene el ID del PC cliente para una sesión activa
func (rss *RemoteSessionService) GetClientPCIDForActiveSession(sessionID string) (string, error) {
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return "", fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	// Verificar que está activa
	if session.Status() != remotesession.StatusActive {
		return "", fmt.Errorf("session is not active")
	}

	return session.ClientPCID(), nil
}

// ValidateStreamingPermission valida que se puede hacer streaming para una sesión
func (rss *RemoteSessionService) ValidateStreamingPermission(sessionID, clientPCID string) error {
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Verificar que está activa
	if session.Status() != remotesession.StatusActive {
		return fmt.Errorf("session is not active")
	}

	// Verificar que el PC cliente coincide
	if session.ClientPCID() != clientPCID {
		return fmt.Errorf("client PC ID mismatch")
	}

	return nil
}

// ValidateInputCommandPermission valida que se puede enviar un comando de input
func (rss *RemoteSessionService) ValidateInputCommandPermission(sessionID, adminUserID string) error {
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Verificar que está activa
	if session.Status() != remotesession.StatusActive {
		return fmt.Errorf("session is not active")
	}

	// Verificar que el administrador coincide
	if session.AdminUserID() != adminUserID {
		return fmt.Errorf("admin user ID mismatch")
	}

	return nil
}

// HandleClientPCDisconnect se encarga de limpiar/finalizar sesiones
// cuando un PC cliente se desconecta.
func (rss *RemoteSessionService) HandleClientPCDisconnect(clientPCID string) error {
	log.Printf("⚡ Handling disconnect for PCID: %s. Checking for active/pending sessions.", clientPCID)
	sessions, err := rss.sessionRepo.FindByClientPCID(clientPCID)
	if err != nil {
		// Si no se encuentran sesiones o hay un error que no sea 'not found',
		// podríamos querer loguearlo pero no necesariamente detener todo.
		// GORM suele devolver gorm.ErrRecordNotFound
		if err.Error() == "record not found" || err.Error() == "sql: no rows in result set" { // Adaptar a los errores específicos de tu repo
			log.Printf("ℹ️ No sessions found for PCID %s during disconnect handling (err: %v). Nothing to clean.", clientPCID, err)
			return nil
		}
		log.Printf("⚠️ Error retrieving sessions for PCID %s during disconnect handling: %v. Proceeding with caution.", clientPCID, err)
		// No retornar aquí, intentar limpiar lo que se pueda si sessions no es nil
	}

	if len(sessions) == 0 {
		log.Printf("ℹ️ No sessions found for PCID %s during disconnect. Nothing to do.", clientPCID)
		return nil
	}

	var sessionsCleanedCount int = 0
	for _, session := range sessions {
		originalStatus := session.Status()
		actionTaken := false
		var newStatusForRepo remotesession.SessionStatus
		var internalErr error

		log.Printf("🔎 Checking session %s for disconnected PCID %s (status: %s)", session.SessionID(), clientPCID, originalStatus)

		if originalStatus == remotesession.StatusActive {
			log.Printf("Ending ACTIVE session %s for disconnected PC %s.", session.SessionID(), clientPCID)
			internalErr = session.End(remotesession.StatusEndedByClient) // O StatusFailed
			if internalErr != nil {
				log.Printf("⚠️ Error calling End(StatusEndedByClient) on session %s: %v. Entity status: %s", session.SessionID(), internalErr, session.Status())
			}
			newStatusForRepo = session.Status()
			actionTaken = true
		} else if originalStatus == remotesession.StatusPendingApproval {
			log.Printf("Rejecting PENDING_APPROVAL session %s for disconnected PC %s.", session.SessionID(), clientPCID)
			internalErr = session.Reject()
			if internalErr != nil {
				log.Printf("⚠️ Error calling Reject() on PENDING_APPROVAL session %s: %v. Entity status: %s", session.SessionID(), internalErr, session.Status())
			}
			newStatusForRepo = session.Status()
			actionTaken = true
		}

		if actionTaken {
			// Solo actualizar si el estado de la entidad realmente cambió o si la operación tuvo éxito
			// (internalErr == nil indica que la operación de cambio de estado en la entidad tuvo éxito)
			if newStatusForRepo != originalStatus || internalErr == nil {
				errUpdate := rss.sessionRepo.UpdateStatus(session.SessionID(), newStatusForRepo)
				if errUpdate != nil {
					log.Printf("❌ CRITICAL: Failed to update session %s status in repo to %s (was %s) during PC disconnect: %v",
						session.SessionID(), newStatusForRepo, originalStatus, errUpdate)
				} else {
					log.Printf("✅ Session %s for disconnected PC %s updated to %s (was %s).",
						session.SessionID(), clientPCID, newStatusForRepo, originalStatus)
					sessionsCleanedCount++

					// Notificar al AdminWeb que la sesión terminó
					if rss.notifySessionEndedCallback != nil {
						log.Printf("📡 Notifying AdminWeb that session %s ended", session.SessionID())
						rss.notifySessionEndedCallback(session.SessionID(), session.ClientPCID(), session.AdminUserID())
					}

					// Aquí podrías emitir eventos de dominio si es necesario
					// event := events.NewRemoteSessionEndedEvent(session.SessionID(), session.AdminUserID(), session.ClientPCID(), string(newStatusForRepo), "Client PC disconnected")
					// rss.eventBus.Publish(event)
				}
			} else {
				log.Printf("ℹ️ Session %s (original status: %s) processed for disconnect, but no state change to repo was made or repo update needed (likely due to internal error during state change: %v). Entity current status: %s",
					session.SessionID(), originalStatus, internalErr, session.Status())
			}
		}
	}
	log.Printf("ℹ️ Finished disconnect handling for PCID %s. Processed %d sessions, cleaned %d sessions that changed state.", clientPCID, len(sessions), sessionsCleanedCount)
	return nil
}

// EndSessionByAdmin finaliza una sesión por parte del administrador
func (rss *RemoteSessionService) EndSessionByAdmin(sessionID string) error {
	// Obtener sesión
	session, err := rss.sessionRepo.FindById(sessionID)
	if err != nil {
		return fmt.Errorf("error finding session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Verificar que está activa
	if session.Status() != remotesession.StatusActive {
		return fmt.Errorf("session is not active")
	}

	// Finalizar sesión usando el método de dominio
	err = session.End(remotesession.StatusEndedByAdmin)
	if err != nil {
		return fmt.Errorf("error ending session: %w", err)
	}

	// Actualizar en repositorio
	err = rss.sessionRepo.UpdateStatus(sessionID, session.Status())
	if err != nil {
		return fmt.Errorf("error updating session status: %w", err)
	}

	// Notificar al AdminWeb que la sesión terminó
	if rss.notifySessionEndedCallback != nil {
		log.Printf("📡 Notifying AdminWeb that session %s ended by admin", sessionID)
		rss.notifySessionEndedCallback(sessionID, session.ClientPCID(), session.AdminUserID())
	}

	// Notificar al cliente que la sesión terminó
	if rss.notifyClientSessionEndedCallback != nil {
		log.Printf("📱 Notifying client %s that session %s ended by admin", session.ClientPCID(), sessionID)
		rss.notifyClientSessionEndedCallback(sessionID, session.ClientPCID())
	}

	log.Printf("✅ Session %s ended by admin", sessionID)
	return nil
}
