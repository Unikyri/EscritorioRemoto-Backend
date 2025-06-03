package entities

import (
	"errors"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/events"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/valueobjects"
	sharedEvents "github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

// ClientPC es la entidad que representa un PC cliente en el dominio
type ClientPC struct {
	// Identificadores
	pcID        *valueobjects.PCID
	identifier  string
	ownerUserID string

	// Estado de conexión
	connectionStatus *valueobjects.ConnectionStatus
	ip               string

	// Timestamps
	registeredAt time.Time
	lastSeenAt   *time.Time
	createdAt    time.Time
	updatedAt    time.Time

	// Domain Events
	domainEvents []sharedEvents.DomainEvent
}

// NewClientPC crea una nueva instancia de ClientPC
func NewClientPC(identifier, ip, ownerUserID string) (*ClientPC, error) {
	// Validaciones de negocio
	if identifier == "" {
		return nil, errors.New("identifier cannot be empty")
	}
	if ip == "" {
		return nil, errors.New("IP address cannot be empty")
	}
	if ownerUserID == "" {
		return nil, errors.New("owner user ID cannot be empty")
	}

	// Crear Value Objects
	pcID := valueobjects.NewPCIDFromUUID()
	connectionStatus := valueobjects.NewOfflineStatus() // PC inicia offline

	now := time.Now().UTC()

	pc := &ClientPC{
		pcID:             pcID,
		identifier:       identifier,
		ownerUserID:      ownerUserID,
		connectionStatus: connectionStatus,
		ip:               ip,
		registeredAt:     now,
		lastSeenAt:       nil,
		createdAt:        now,
		updatedAt:        now,
		domainEvents:     make([]sharedEvents.DomainEvent, 0),
	}

	return pc, nil
}

// GETTERS
func (pc *ClientPC) ID() *valueobjects.PCID {
	return pc.pcID
}

func (pc *ClientPC) Identifier() string {
	return pc.identifier
}

func (pc *ClientPC) OwnerUserID() string {
	return pc.ownerUserID
}

func (pc *ClientPC) ConnectionStatus() *valueobjects.ConnectionStatus {
	return pc.connectionStatus
}

func (pc *ClientPC) IP() string {
	return pc.ip
}

func (pc *ClientPC) RegisteredAt() time.Time {
	return pc.registeredAt
}

func (pc *ClientPC) LastSeenAt() *time.Time {
	return pc.lastSeenAt
}

func (pc *ClientPC) CreatedAt() time.Time {
	return pc.createdAt
}

func (pc *ClientPC) UpdatedAt() time.Time {
	return pc.updatedAt
}

// COMPORTAMIENTOS DE NEGOCIO

// SetOnline marca el PC como online y genera eventos
func (pc *ClientPC) SetOnline() error {
	newStatus := valueobjects.NewOnlineStatus()

	// Verificar si la transición es válida
	if err := pc.connectionStatus.CanTransitionTo(newStatus); err != nil {
		return err
	}

	oldStatus := pc.connectionStatus
	pc.connectionStatus = newStatus
	pc.lastSeenAt = timePointer(time.Now().UTC())
	pc.updatedAt = time.Now().UTC()

	// Generar evento solo si cambió el estado
	if !oldStatus.IsOnline() {
		event := events.NewPCConnectedEvent(
			pc.pcID.Value(),
			pc.identifier,
			pc.ip,
			pc.ownerUserID,
			*pc.lastSeenAt,
		)
		pc.addDomainEvent(event)
	}

	return nil
}

// SetOffline marca el PC como offline y genera eventos
func (pc *ClientPC) SetOffline(reason string) error {
	newStatus := valueobjects.NewOfflineStatus()

	// Verificar si la transición es válida
	if err := pc.connectionStatus.CanTransitionTo(newStatus); err != nil {
		return err
	}

	oldStatus := pc.connectionStatus
	pc.connectionStatus = newStatus
	pc.updatedAt = time.Now().UTC()

	// Generar evento solo si cambió el estado
	if !oldStatus.IsOffline() {
		event := events.NewPCDisconnectedEvent(
			pc.pcID.Value(),
			pc.identifier,
			pc.ownerUserID,
			time.Now().UTC(),
			reason,
		)
		pc.addDomainEvent(event)
	}

	return nil
}

// SetConnecting marca el PC como conectando
func (pc *ClientPC) SetConnecting() error {
	newStatus := valueobjects.NewConnectingStatus()

	// Verificar si la transición es válida
	if err := pc.connectionStatus.CanTransitionTo(newStatus); err != nil {
		return err
	}

	pc.connectionStatus = newStatus
	pc.updatedAt = time.Now().UTC()

	return nil
}

// UpdateLastSeen actualiza el timestamp de última actividad
func (pc *ClientPC) UpdateLastSeen() {
	now := time.Now().UTC()
	pc.lastSeenAt = &now
	pc.updatedAt = now
}

// UpdateIP actualiza la dirección IP del PC
func (pc *ClientPC) UpdateIP(newIP string) error {
	if newIP == "" {
		return errors.New("IP address cannot be empty")
	}

	pc.ip = newIP
	pc.updatedAt = time.Now().UTC()

	return nil
}

// MÉTODOS DE CONSULTA

// IsOnline verifica si el PC está online
func (pc *ClientPC) IsOnline() bool {
	return pc.connectionStatus.IsOnline()
}

// IsOffline verifica si el PC está offline
func (pc *ClientPC) IsOffline() bool {
	return pc.connectionStatus.IsOffline()
}

// IsConnecting verifica si el PC está conectando
func (pc *ClientPC) IsConnecting() bool {
	return pc.connectionStatus.IsConnecting()
}

// IsConnectionExpired verifica si la conexión ha expirado
func (pc *ClientPC) IsConnectionExpired(timeout time.Duration) bool {
	if pc.lastSeenAt == nil {
		return true
	}

	return time.Since(*pc.lastSeenAt) > timeout
}

// DOMAIN EVENTS

// DomainEvents retorna los eventos de dominio pendientes
func (pc *ClientPC) DomainEvents() []sharedEvents.DomainEvent {
	return pc.domainEvents
}

// ClearDomainEvents limpia los eventos de dominio
func (pc *ClientPC) ClearDomainEvents() {
	pc.domainEvents = make([]sharedEvents.DomainEvent, 0)
}

// addDomainEvent agrega un evento de dominio
func (pc *ClientPC) addDomainEvent(event sharedEvents.DomainEvent) {
	pc.domainEvents = append(pc.domainEvents, event)
}

// MÉTODOS DE RECONSTRUCCIÓN (para repositorio)

// ReconstructFromDB reconstruye la entidad desde datos de base de datos
func ReconstructClientPCFromDB(
	pcIDStr, identifier, ownerUserID, connectionStatusStr, ip string,
	registeredAt, createdAt, updatedAt time.Time,
	lastSeenAt *time.Time,
) (*ClientPC, error) {

	// Reconstruir Value Objects
	pcID, err := valueobjects.NewPCID(pcIDStr)
	if err != nil {
		return nil, err
	}

	connectionStatus, err := valueobjects.NewConnectionStatus(connectionStatusStr)
	if err != nil {
		return nil, err
	}

	pc := &ClientPC{
		pcID:             pcID,
		identifier:       identifier,
		ownerUserID:      ownerUserID,
		connectionStatus: connectionStatus,
		ip:               ip,
		registeredAt:     registeredAt,
		lastSeenAt:       lastSeenAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		domainEvents:     make([]sharedEvents.DomainEvent, 0),
	}

	return pc, nil
}

// timePointer es una función helper para crear punteros a time.Time
func timePointer(t time.Time) *time.Time {
	return &t
}
