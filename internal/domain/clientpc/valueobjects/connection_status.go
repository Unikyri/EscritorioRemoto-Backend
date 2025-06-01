package valueobjects

import (
	"errors"
	"fmt"
)

// ConnectionStatus es un Value Object que representa el estado de conexión de un PC
type ConnectionStatus struct {
	value string
}

// Constantes para los estados válidos
const (
	StatusOnline     = "ONLINE"
	StatusOffline    = "OFFLINE"
	StatusConnecting = "CONNECTING"
)

// NewConnectionStatus crea un nuevo ConnectionStatus validado
func NewConnectionStatus(value string) (*ConnectionStatus, error) {
	if !isValidStatus(value) {
		return nil, fmt.Errorf("invalid connection status: %s. Valid values are: %s, %s, %s",
			value, StatusOnline, StatusOffline, StatusConnecting)
	}

	return &ConnectionStatus{value: value}, nil
}

// NewOnlineStatus crea un ConnectionStatus en estado ONLINE
func NewOnlineStatus() *ConnectionStatus {
	return &ConnectionStatus{value: StatusOnline}
}

// NewOfflineStatus crea un ConnectionStatus en estado OFFLINE
func NewOfflineStatus() *ConnectionStatus {
	return &ConnectionStatus{value: StatusOffline}
}

// NewConnectingStatus crea un ConnectionStatus en estado CONNECTING
func NewConnectingStatus() *ConnectionStatus {
	return &ConnectionStatus{value: StatusConnecting}
}

// Value retorna el valor del estado
func (cs *ConnectionStatus) Value() string {
	return cs.value
}

// String implementa fmt.Stringer
func (cs *ConnectionStatus) String() string {
	return cs.value
}

// IsOnline verifica si el estado es ONLINE
func (cs *ConnectionStatus) IsOnline() bool {
	return cs.value == StatusOnline
}

// IsOffline verifica si el estado es OFFLINE
func (cs *ConnectionStatus) IsOffline() bool {
	return cs.value == StatusOffline
}

// IsConnecting verifica si el estado es CONNECTING
func (cs *ConnectionStatus) IsConnecting() bool {
	return cs.value == StatusConnecting
}

// Equals compara dos ConnectionStatus
func (cs *ConnectionStatus) Equals(other *ConnectionStatus) bool {
	if other == nil {
		return false
	}
	return cs.value == other.value
}

// CanTransitionTo verifica si es válida la transición a otro estado
func (cs *ConnectionStatus) CanTransitionTo(newStatus *ConnectionStatus) error {
	if newStatus == nil {
		return errors.New("new status cannot be nil")
	}

	// Definir transiciones válidas
	validTransitions := map[string][]string{
		StatusOffline:    {StatusConnecting, StatusOnline},
		StatusConnecting: {StatusOnline, StatusOffline},
		StatusOnline:     {StatusOffline, StatusConnecting},
	}

	allowedStates, exists := validTransitions[cs.value]
	if !exists {
		return fmt.Errorf("unknown current status: %s", cs.value)
	}

	for _, allowed := range allowedStates {
		if newStatus.value == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", cs.value, newStatus.value)
}

// isValidStatus verifica si un string es un estado válido
func isValidStatus(status string) bool {
	validStatuses := []string{StatusOnline, StatusOffline, StatusConnecting}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}
