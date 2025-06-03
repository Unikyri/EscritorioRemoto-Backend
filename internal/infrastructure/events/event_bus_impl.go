package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

// EventBusImpl implementa el patrón Observer para eventos de dominio
type EventBusImpl struct {
	handlers map[string][]events.EventHandler
	mutex    sync.RWMutex
}

// NewEventBus crea una nueva instancia del bus de eventos
func NewEventBus() events.EventBus {
	return &EventBusImpl{
		handlers: make(map[string][]events.EventHandler),
		mutex:    sync.RWMutex{},
	}
}

// Publish publica un evento para todos los handlers interesados
func (eb *EventBusImpl) Publish(ctx context.Context, event events.DomainEvent) error {
	eb.mutex.RLock()
	handlers, exists := eb.handlers[event.Type()]
	eb.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		// No hay handlers para este tipo de evento, no es un error
		return nil
	}

	// Ejecutar handlers de forma asíncrona para no bloquear
	for _, handler := range handlers {
		go func(h events.EventHandler) {
			if err := h.Handle(ctx, event); err != nil {
				// En una implementación real, aquí iría logging
				fmt.Printf("Error handling event %s: %v\n", event.Type(), err)
			}
		}(handler)
	}

	return nil
}

// Subscribe registra un handler para tipos específicos de eventos
func (eb *EventBusImpl) Subscribe(handler events.EventHandler, eventTypes ...string) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	if len(eventTypes) == 0 {
		return fmt.Errorf("at least one event type must be specified")
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	for _, eventType := range eventTypes {
		if eventType == "" {
			continue
		}

		// Verificar que el handler puede manejar este tipo de evento
		if !handler.CanHandle(eventType) {
			continue
		}

		// Agregar handler si no existe ya
		if !eb.handlerExists(eventType, handler) {
			eb.handlers[eventType] = append(eb.handlers[eventType], handler)
		}
	}

	return nil
}

// Unsubscribe desregistra un handler
func (eb *EventBusImpl) Unsubscribe(handler events.EventHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	// Remover handler de todos los tipos de eventos
	for eventType, handlers := range eb.handlers {
		newHandlers := make([]events.EventHandler, 0)
		for _, h := range handlers {
			if h != handler {
				newHandlers = append(newHandlers, h)
			}
		}
		eb.handlers[eventType] = newHandlers
	}

	return nil
}

// Close cierra el bus de eventos y limpia recursos
func (eb *EventBusImpl) Close() error {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	// Limpiar todos los handlers
	eb.handlers = make(map[string][]events.EventHandler)

	return nil
}

// handlerExists verifica si un handler ya está registrado para un tipo de evento
func (eb *EventBusImpl) handlerExists(eventType string, handler events.EventHandler) bool {
	handlers, exists := eb.handlers[eventType]
	if !exists {
		return false
	}

	for _, h := range handlers {
		if h == handler {
			return true
		}
	}

	return false
}

// GetHandlerCount retorna el número de handlers registrados para un tipo de evento
func (eb *EventBusImpl) GetHandlerCount(eventType string) int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	handlers, exists := eb.handlers[eventType]
	if !exists {
		return 0
	}

	return len(handlers)
}
