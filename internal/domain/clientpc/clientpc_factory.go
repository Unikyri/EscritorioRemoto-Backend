package clientpc

import (
	"github.com/google/uuid"
)

// IClientPCFactory defines the interface for creating ClientPC instances
type IClientPCFactory interface {
	CreateClientPC(identifier, ip, ownerUserID string) (*ClientPC, error)
}

// ClientPCFactory implements the Factory pattern for ClientPC creation
type ClientPCFactory struct{}

// NewClientPCFactory creates a new instance of ClientPCFactory
func NewClientPCFactory() IClientPCFactory {
	return &ClientPCFactory{}
}

// CreateClientPC creates a new ClientPC with generated UUID
func (f *ClientPCFactory) CreateClientPC(identifier, ip, ownerUserID string) (*ClientPC, error) {
	// Generate a new UUID for the PC
	pcID := uuid.New().String()

	// Create the ClientPC using the domain constructor
	return NewClientPC(pcID, identifier, ip, ownerUserID)
}
