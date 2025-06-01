package repositories

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/entities"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/valueobjects"
)

// IClientPCRepository define la interface del repositorio de Client PCs
// Esta interface está en el domain para seguir el Dependency Inversion Principle
type IClientPCRepository interface {
	// Save guarda o actualiza un ClientPC
	Save(ctx context.Context, pc *entities.ClientPC) error

	// FindByID busca un ClientPC por su ID
	FindByID(ctx context.Context, pcID *valueobjects.PCID) (*entities.ClientPC, error)

	// FindByIdentifierAndOwner busca un PC por identificador y propietario
	FindByIdentifierAndOwner(ctx context.Context, identifier, ownerUserID string) (*entities.ClientPC, error)

	// FindByOwner busca todos los PCs de un propietario
	FindByOwner(ctx context.Context, ownerUserID string) ([]*entities.ClientPC, error)

	// FindOnlineByOwner busca todos los PCs online de un propietario
	FindOnlineByOwner(ctx context.Context, ownerUserID string) ([]*entities.ClientPC, error)

	// FindAll busca todos los ClientPCs con paginación
	FindAll(ctx context.Context, limit, offset int) ([]*entities.ClientPC, error)

	// FindOnlineClientPCs busca todos los PCs que están online
	FindOnlineClientPCs(ctx context.Context) ([]*entities.ClientPC, error)

	// UpdateConnectionStatus actualiza solo el estado de conexión
	UpdateConnectionStatus(ctx context.Context, pcID *valueobjects.PCID, status *valueobjects.ConnectionStatus) error

	// UpdateLastSeen actualiza el timestamp de última conexión
	UpdateLastSeen(ctx context.Context, pcID *valueobjects.PCID) error

	// Delete elimina un ClientPC (para casos excepcionales)
	Delete(ctx context.Context, pcID *valueobjects.PCID) error

	// Count retorna el número total de ClientPCs
	Count(ctx context.Context) (int64, error)

	// CountOnline retorna el número de ClientPCs online
	CountOnline(ctx context.Context) (int64, error)
}
