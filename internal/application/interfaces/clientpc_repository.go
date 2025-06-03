package interfaces

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// IClientPCRepository defines the interface for ClientPC data persistence operations
type IClientPCRepository interface {
	// Save stores a new ClientPC or updates an existing one
	Save(ctx context.Context, pc *clientpc.ClientPC) error

	// FindByID retrieves a ClientPC by its ID
	FindByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error)

	// FindByIdentifierAndOwner retrieves a ClientPC by identifier and owner user ID
	FindByIdentifierAndOwner(ctx context.Context, identifier string, ownerID string) (*clientpc.ClientPC, error)

	// FindByOwner retrieves all ClientPCs belonging to a specific owner
	FindByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error)

	// FindOnlineByOwner retrieves all online ClientPCs belonging to a specific owner
	FindOnlineByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error)

	// UpdateConnectionStatus updates the connection status of a ClientPC
	UpdateConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error

	// UpdateLastSeen updates the last seen timestamp of a ClientPC
	UpdateLastSeen(ctx context.Context, pcID string) error

	// Delete removes a ClientPC from the repository
	Delete(ctx context.Context, pcID string) error

	// FindAll retrieves all ClientPCs with optional filtering
	FindAll(ctx context.Context, limit, offset int) ([]*clientpc.ClientPC, error)

	// CountByOwner returns the count of ClientPCs for a specific owner
	CountByOwner(ctx context.Context, ownerID string) (int, error)
}
