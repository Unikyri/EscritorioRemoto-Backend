package events

import (
	"context"
)

// EventHandler define un manejador de eventos
type EventHandler interface {
	// Handle procesa un evento de dominio
	Handle(ctx context.Context, event DomainEvent) error

	// CanHandle verifica si puede manejar un tipo de evento
	CanHandle(eventType string) bool
}

// EventBus define la interface para el bus de eventos (patrón Observer)
type EventBus interface {
	// Publish publica un evento para todos los handlers interesados
	Publish(ctx context.Context, event DomainEvent) error

	// Subscribe registra un handler para tipos específicos de eventos
	Subscribe(handler EventHandler, eventTypes ...string) error

	// Unsubscribe desregistra un handler
	Unsubscribe(handler EventHandler) error

	// Close cierra el bus de eventos y limpia recursos
	Close() error
}

// EventDispatcher define un despachador de eventos para agregados
type EventDispatcher interface {
	// Dispatch despacha todos los eventos pendientes de un agregado
	Dispatch(ctx context.Context, events []DomainEvent) error
}
