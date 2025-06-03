package events

import "github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"

// IEventBus interface para el bus de eventos
type IEventBus interface {
	Publish(event events.DomainEvent)
	Subscribe(eventType string, handler EventHandler)
}

// EventHandler interface para manejadores de eventos
type EventHandler interface {
	Handle(event events.DomainEvent) error
}

// SimpleEventBus implementación simple del bus de eventos
type SimpleEventBus struct {
	handlers map[string][]EventHandler
}

// NewSimpleEventBus crea una nueva instancia del bus de eventos
func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publica un evento
func (bus *SimpleEventBus) Publish(event events.DomainEvent) {
	if handlers, exists := bus.handlers[event.Type()]; exists {
		for _, handler := range handlers {
			// En una implementación real, esto sería asíncrono
			go func(h EventHandler) {
				_ = h.Handle(event)
			}(handler)
		}
	}
}

// Subscribe suscribe un manejador a un tipo de evento
func (bus *SimpleEventBus) Subscribe(eventType string, handler EventHandler) {
	if _, exists := bus.handlers[eventType]; !exists {
		bus.handlers[eventType] = make([]EventHandler, 0)
	}
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}
