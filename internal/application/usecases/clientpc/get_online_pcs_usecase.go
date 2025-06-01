package clientpc

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/entities"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/repositories"
)

// GetOnlinePCsRequest define los datos de entrada para el use case
type GetOnlinePCsRequest struct {
	// En este caso no necesitamos parámetros especiales
	// pero mantenemos la estructura para consistencia
}

// GetOnlinePCsResponse define los datos de salida del use case
type GetOnlinePCsResponse struct {
	PCs   []*entities.ClientPC `json:"pcs"`
	Count int                  `json:"count"`
}

// IGetOnlinePCsUseCase define la interface del use case
type IGetOnlinePCsUseCase interface {
	Execute(ctx context.Context, request GetOnlinePCsRequest) (*GetOnlinePCsResponse, error)
}

// GetOnlinePCsUseCase implementa el caso de uso para obtener PCs online
type GetOnlinePCsUseCase struct {
	pcRepository repositories.IClientPCRepository
}

// NewGetOnlinePCsUseCase crea una nueva instancia del use case
func NewGetOnlinePCsUseCase(pcRepository repositories.IClientPCRepository) IGetOnlinePCsUseCase {
	return &GetOnlinePCsUseCase{
		pcRepository: pcRepository,
	}
}

// Execute ejecuta el caso de uso para obtener PCs online
func (uc *GetOnlinePCsUseCase) Execute(ctx context.Context, request GetOnlinePCsRequest) (*GetOnlinePCsResponse, error) {
	// 1. Obtener todos los PCs online
	pcs, err := uc.pcRepository.FindOnlineClientPCs(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Si no hay PCs, retornar slice vacío (no nil)
	if pcs == nil {
		pcs = make([]*entities.ClientPC, 0)
	}

	// 3. Construir respuesta
	return &GetOnlinePCsResponse{
		PCs:   pcs,
		Count: len(pcs),
	}, nil
}
