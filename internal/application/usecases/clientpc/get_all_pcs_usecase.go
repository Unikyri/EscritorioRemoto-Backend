package clientpc

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// GetAllPCsRequest define los datos de entrada para el use case
type GetAllPCsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// GetAllPCsResponse define los datos de salida del use case
type GetAllPCsResponse struct {
	PCs    []*clientpc.ClientPC `json:"pcs"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// IGetAllPCsUseCase define la interface del use case
type IGetAllPCsUseCase interface {
	Execute(ctx context.Context, request GetAllPCsRequest) (*GetAllPCsResponse, error)
}

// GetAllPCsUseCase implementa el caso de uso para obtener todos los PCs
type GetAllPCsUseCase struct {
	pcRepository interfaces.IClientPCRepository
}

// NewGetAllPCsUseCase crea una nueva instancia del use case
func NewGetAllPCsUseCase(pcRepository interfaces.IClientPCRepository) IGetAllPCsUseCase {
	return &GetAllPCsUseCase{
		pcRepository: pcRepository,
	}
}

// Execute ejecuta el caso de uso para obtener todos los PCs
func (uc *GetAllPCsUseCase) Execute(ctx context.Context, request GetAllPCsRequest) (*GetAllPCsResponse, error) {
	// 1. Validar parámetros de paginación
	limit := request.Limit
	offset := request.Offset

	if limit <= 0 {
		limit = 50 // Límite por defecto
	}
	if limit > 100 {
		limit = 100 // Límite máximo
	}
	if offset < 0 {
		offset = 0
	}

	// 2. Obtener PCs con paginación
	pcs, err := uc.pcRepository.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// 3. Para paginación, usar la cantidad de resultados obtenidos como total aproximado
	// En una implementación más completa, podríamos agregar un método Count a la interface
	total := int64(len(pcs))
	if len(pcs) == limit {
		// Si obtuvimos el límite completo, probablemente hay más
		total = int64(offset + limit + 1) // Estimación conservadora
	}

	// 4. Construir respuesta
	return &GetAllPCsResponse{
		PCs:    pcs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
