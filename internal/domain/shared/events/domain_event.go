package events

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent define la interface base para todos los eventos de dominio
type DomainEvent interface {
	// ID retorna el identificador único del evento
	ID() string

	// Type retorna el tipo del evento
	Type() string

	// OccurredOn retorna el timestamp cuando ocurrió el evento
	OccurredOn() time.Time

	// AggregateID retorna el ID del agregado que generó el evento
	AggregateID() string

	// Version retorna la versión del evento para compatibilidad
	Version() int

	// Data retorna los datos específicos del evento
	Data() interface{}
}

// BaseDomainEvent es la implementación base para eventos de dominio
type BaseDomainEvent struct {
	id          string
	eventType   string
	occurredOn  time.Time
	aggregateID string
	version     int
	data        interface{}
}

// NewBaseDomainEvent crea un nuevo evento base
func NewBaseDomainEvent(eventType, aggregateID string, data interface{}) *BaseDomainEvent {
	return &BaseDomainEvent{
		id:          uuid.New().String(),
		eventType:   eventType,
		occurredOn:  time.Now().UTC(),
		aggregateID: aggregateID,
		version:     1,
		data:        data,
	}
}

// ID implementa DomainEvent.ID
func (e *BaseDomainEvent) ID() string {
	return e.id
}

// Type implementa DomainEvent.Type
func (e *BaseDomainEvent) Type() string {
	return e.eventType
}

// OccurredOn implementa DomainEvent.OccurredOn
func (e *BaseDomainEvent) OccurredOn() time.Time {
	return e.occurredOn
}

// AggregateID implementa DomainEvent.AggregateID
func (e *BaseDomainEvent) AggregateID() string {
	return e.aggregateID
}

// Version implementa DomainEvent.Version
func (e *BaseDomainEvent) Version() int {
	return e.version
}

// Data implementa DomainEvent.Data
func (e *BaseDomainEvent) Data() interface{} {
	return e.data
}
