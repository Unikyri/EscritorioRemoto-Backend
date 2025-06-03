package clientpc

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// GetOnlinePCsRequest define los datos de entrada para el use case
type GetOnlinePCsRequest struct {
	// En este caso no necesitamos parámetros especiales
	// pero mantenemos la estructura para consistencia
}

// GetOnlinePCsResponse define los datos de salida del use case
type GetOnlinePCsResponse struct {
	PCs   []*clientpc.ClientPC `json:"pcs"`
	Count int                  `json:"count"`
}

// IGetOnlinePCsUseCase define la interface del use case
type IGetOnlinePCsUseCase interface {
	Execute(ctx context.Context, request GetOnlinePCsRequest) (*GetOnlinePCsResponse, error)
}

// GetOnlinePCsUseCase implementa el caso de uso para obtener PCs online
type GetOnlinePCsUseCase struct {
	pcRepository interfaces.IClientPCRepository
}

// NewGetOnlinePCsUseCase crea una nueva instancia del use case
func NewGetOnlinePCsUseCase(pcRepository interfaces.IClientPCRepository) IGetOnlinePCsUseCase {
	return &GetOnlinePCsUseCase{
		pcRepository: pcRepository,
	}
}

// Execute ejecuta el caso de uso para obtener PCs online
func (uc *GetOnlinePCsUseCase) Execute(ctx context.Context, request GetOnlinePCsRequest) (*GetOnlinePCsResponse, error) {
	// 1. Obtener todos los PCs y filtrar los que están online
	allPCs, err := uc.pcRepository.FindAll(ctx, 0, 0) // Sin límite para obtener todos
	if err != nil {
		return nil, err
	}

	// 2. Filtrar solo los PCs online
	onlinePCs := make([]*clientpc.ClientPC, 0)
	for _, pc := range allPCs {
		if pc.IsOnline() {
			onlinePCs = append(onlinePCs, pc)
		}
	}

	// 3. Construir respuesta
	return &GetOnlinePCsResponse{
		PCs:   onlinePCs,
		Count: len(onlinePCs),
	}, nil
}
