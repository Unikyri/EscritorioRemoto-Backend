package events

import (
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

const (
	// PCDisconnectedEventType es el tipo del evento de desconexión de PC
	PCDisconnectedEventType = "pc.disconnected"
)

// PCDisconnectedEventData contiene los datos específicos del evento
type PCDisconnectedEventData struct {
	PCID           string    `json:"pc_id"`
	Identifier     string    `json:"identifier"`
	OwnerUserID    string    `json:"owner_user_id"`
	DisconnectedAt time.Time `json:"disconnected_at"`
	Reason         string    `json:"reason,omitempty"`
}

// PCDisconnectedEvent es el evento que se dispara cuando un PC se desconecta
type PCDisconnectedEvent struct {
	*events.BaseDomainEvent
	pcData *PCDisconnectedEventData
}

// NewPCDisconnectedEvent crea un nuevo evento de PC desconectado
func NewPCDisconnectedEvent(pcID, identifier, ownerUserID string, disconnectedAt time.Time, reason string) *PCDisconnectedEvent {
	data := &PCDisconnectedEventData{
		PCID:           pcID,
		Identifier:     identifier,
		OwnerUserID:    ownerUserID,
		DisconnectedAt: disconnectedAt,
		Reason:         reason,
	}

	baseEvent := events.NewBaseDomainEvent(PCDisconnectedEventType, pcID, data)

	return &PCDisconnectedEvent{
		BaseDomainEvent: baseEvent,
		pcData:          data,
	}
}

// GetPCData retorna los datos específicos del PC desconectado
func (e *PCDisconnectedEvent) GetPCData() *PCDisconnectedEventData {
	return e.pcData
}
