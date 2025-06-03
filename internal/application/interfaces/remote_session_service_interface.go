package interfaces

import (
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/remotesession"
)

// IRemoteSessionService define la interfaz para el servicio de sesiones remotas
type IRemoteSessionService interface {
	InitiateSession(adminUserID, clientPCID string) (*remotesession.RemoteSession, error)
	AcceptSession(sessionID string) error
	RejectSession(sessionID, reason string) error
	GetSessionById(sessionID string) (*remotesession.RemoteSession, error)
	GetActiveSessions() ([]*remotesession.RemoteSession, error)
	GetSessionsByUser(userID string) ([]*remotesession.RemoteSession, error)
	GetSessionsByPC(clientPCID string) ([]*remotesession.RemoteSession, error)
	CleanupStuckSessions(clientPCID string) error
	GetActiveSessionForPC(clientPCID string) (*remotesession.RemoteSession, error)
	IsSessionActiveForStreaming(sessionID string) (bool, error)
	GetAdminUserIDForActiveSession(sessionID string) (string, error)
	GetClientPCIDForActiveSession(sessionID string) (string, error)
	ValidateStreamingPermission(sessionID, clientPCID string) error
	ValidateInputCommandPermission(sessionID, adminUserID string) error
	HandleClientPCDisconnect(clientPCID string) error
} 