package events

import (
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

const (
	// PCConnectedEventType es el tipo del evento de conexión de PC
	PCConnectedEventType = "pc.connected"
)

// PCConnectedEventData contiene los datos específicos del evento
type PCConnectedEventData struct {
	PCID        string    `json:"pc_id"`
	Identifier  string    `json:"identifier"`
	IP          string    `json:"ip"`
	OwnerUserID string    `json:"owner_user_id"`
	ConnectedAt time.Time `json:"connected_at"`
}

// PCConnectedEvent es el evento que se dispara cuando un PC se conecta
type PCConnectedEvent struct {
	*events.BaseDomainEvent
	pcData *PCConnectedEventData
}

// NewPCConnectedEvent crea un nuevo evento de PC conectado
func NewPCConnectedEvent(pcID, identifier, ip, ownerUserID string, connectedAt time.Time) *PCConnectedEvent {
	data := &PCConnectedEventData{
		PCID:        pcID,
		Identifier:  identifier,
		IP:          ip,
		OwnerUserID: ownerUserID,
		ConnectedAt: connectedAt,
	}

	baseEvent := events.NewBaseDomainEvent(PCConnectedEventType, pcID, data)

	return &PCConnectedEvent{
		BaseDomainEvent: baseEvent,
		pcData:          data,
	}
}

// GetPCData retorna los datos específicos del PC conectado
func (e *PCConnectedEvent) GetPCData() *PCConnectedEventData {
	return e.pcData
}

// PCConnectionData es un alias para compatibilidad
type PCConnectionData = PCConnectedEventData
