package di

import (
	"database/sql"

	// Application Layer
	"github.com/unikyri/escritorio-remoto-backend/internal/application/usecases/clientpc"

	// Domain Layer
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/repositories"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/services"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"

	// Infrastructure Layer
	infraEvents "github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/events"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/observers"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/persistence/mysql"
)

// Container contiene todas las dependencias del sistema
type Container struct {
	// Database
	DB *sql.DB

	// Domain Services
	PCDomainService services.IPCDomainService

	// Repositories
	ClientPCRepository repositories.IClientPCRepository

	// Event System
	EventBus              events.EventBus
	WebSocketEventHandler *observers.WebSocketEventHandler

	// Use Cases
	RegisterPCUseCase   clientpc.IRegisterPCUseCase
	GetAllPCsUseCase    clientpc.IGetAllPCsUseCase
	GetOnlinePCsUseCase clientpc.IGetOnlinePCsUseCase
}

// NewContainer crea un nuevo contenedor de dependencias
func NewContainer(db *sql.DB) *Container {
	container := &Container{
		DB: db,
	}

	container.registerDomainServices()
	container.registerRepositories()
	container.registerEventSystem()
	container.registerUseCases()

	return container
}

// registerDomainServices registra los servicios de dominio
func (c *Container) registerDomainServices() {
	c.PCDomainService = services.NewPCDomainService()
}

// registerRepositories registra las implementaciones de repositorios
func (c *Container) registerRepositories() {
	c.ClientPCRepository = mysql.NewClientPCRepository(c.DB)
}

// registerEventSystem registra el sistema de eventos (patrón Observer)
func (c *Container) registerEventSystem() {
	// Event Bus
	c.EventBus = infraEvents.NewEventBus()

	// Event Handlers (Observers)
	c.WebSocketEventHandler = observers.NewWebSocketEventHandler()

	// Registrar handlers en el Event Bus
	c.EventBus.Subscribe(
		c.WebSocketEventHandler,
		"pc.connected",
		"pc.disconnected",
	)
}

// registerUseCases registra los casos de uso de la aplicación
func (c *Container) registerUseCases() {
	c.RegisterPCUseCase = clientpc.NewRegisterPCUseCase(
		c.ClientPCRepository,
		c.PCDomainService,
		c.EventBus,
	)

	c.GetAllPCsUseCase = clientpc.NewGetAllPCsUseCase(
		c.ClientPCRepository,
	)

	c.GetOnlinePCsUseCase = clientpc.NewGetOnlinePCsUseCase(
		c.ClientPCRepository,
	)
}

// Close cierra todas las conexiones y limpia recursos
func (c *Container) Close() error {
	if c.EventBus != nil {
		c.EventBus.Close()
	}

	if c.DB != nil {
		return c.DB.Close()
	}

	return nil
}
