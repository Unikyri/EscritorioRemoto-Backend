package clientpc

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/entities"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/repositories"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/services"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/events"
)

// RegisterPCRequest define los datos de entrada para el use case
type RegisterPCRequest struct {
	OwnerUserID string `json:"owner_user_id" validate:"required"`
	Identifier  string `json:"identifier" validate:"required"`
	IP          string `json:"ip" validate:"required"`
}

// RegisterPCResponse define los datos de salida del use case
type RegisterPCResponse struct {
	PC      *entities.ClientPC `json:"pc"`
	IsNew   bool               `json:"is_new"`
	Message string             `json:"message"`
}

// IRegisterPCUseCase define la interface del use case
type IRegisterPCUseCase interface {
	Execute(ctx context.Context, request RegisterPCRequest) (*RegisterPCResponse, error)
}

// RegisterPCUseCase implementa el caso de uso para registrar un PC
type RegisterPCUseCase struct {
	pcRepository    repositories.IClientPCRepository
	pcDomainService services.IPCDomainService
	eventBus        events.EventBus
}

// NewRegisterPCUseCase crea una nueva instancia del use case
func NewRegisterPCUseCase(
	pcRepository repositories.IClientPCRepository,
	pcDomainService services.IPCDomainService,
	eventBus events.EventBus,
) IRegisterPCUseCase {
	return &RegisterPCUseCase{
		pcRepository:    pcRepository,
		pcDomainService: pcDomainService,
		eventBus:        eventBus,
	}
}

// Execute ejecuta el caso de uso para registrar un PC
func (uc *RegisterPCUseCase) Execute(ctx context.Context, request RegisterPCRequest) (*RegisterPCResponse, error) {
	// 1. Buscar PC existente
	existingPC, err := uc.pcRepository.FindByIdentifierAndOwner(ctx, request.Identifier, request.OwnerUserID)
	if err != nil {
		return nil, err
	}

	// 2. Usar Domain Service para determinar la acción
	isNew := uc.pcDomainService.ShouldCreateNewPC(existingPC, request.Identifier, request.OwnerUserID)

	// 3. Procesar con Domain Service
	pc, err := uc.pcDomainService.RegisterOrUpdatePC(existingPC, request.Identifier, request.IP, request.OwnerUserID)
	if err != nil {
		return nil, err
	}

	// 4. Persistir la entidad
	if err := uc.pcRepository.Save(ctx, pc); err != nil {
		return nil, err
	}

	// 5. Publicar eventos de dominio
	if len(pc.DomainEvents()) > 0 {
		for _, event := range pc.DomainEvents() {
			if err := uc.eventBus.Publish(ctx, event); err != nil {
				// Log error but don't fail the operation
				// En una implementación real, aquí iría un logger
			}
		}
		pc.ClearDomainEvents()
	}

	// 6. Construir respuesta
	message := "PC updated successfully"
	if isNew {
		message = "PC registered successfully"
	}

	return &RegisterPCResponse{
		PC:      pc,
		IsNew:   isNew,
		Message: message,
	}, nil
}
