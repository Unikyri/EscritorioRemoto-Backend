package pcservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// IPCService defines the interface for PC-related business operations
type IPCService interface {
	RegisterPC(ctx context.Context, ownerUserID, pcIdentifier, ip string) (*clientpc.ClientPC, error)
	GetPCByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error)
	GetPCsByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error)
	GetOnlinePCsByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error)
	UpdatePCConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error
	UpdatePCLastSeen(ctx context.Context, pcID string) error
	GetAllClientPCs(ctx context.Context) ([]*clientpc.ClientPC, error)
	GetOnlineClientPCs(ctx context.Context) ([]*clientpc.ClientPC, error)
}

// PCService implements the business logic for PC operations
type PCService struct {
	pcRepository interfaces.IClientPCRepository
	pcFactory    clientpc.IClientPCFactory
}

// NewPCService creates a new instance of PCService
func NewPCService(pcRepository interfaces.IClientPCRepository, pcFactory clientpc.IClientPCFactory) IPCService {
	return &PCService{
		pcRepository: pcRepository,
		pcFactory:    pcFactory,
	}
}

// RegisterPC registers a new PC or updates an existing one for a specific user
func (s *PCService) RegisterPC(ctx context.Context, ownerUserID, pcIdentifier, ip string) (*clientpc.ClientPC, error) {
	fmt.Printf("DEBUG RegisterPC: Starting registration for user=%s, identifier=%s, ip=%s\n", ownerUserID, pcIdentifier, ip)
	
	// Validate input parameters
	if ownerUserID == "" {
		return nil, errors.New("owner user ID cannot be empty")
	}
	if pcIdentifier == "" {
		return nil, errors.New("PC identifier cannot be empty")
	}
	if ip == "" {
		return nil, errors.New("IP address cannot be empty")
	}

	fmt.Printf("DEBUG RegisterPC: Input validation passed\n")

	// Check if PC is already registered for this user
	existingPC, err := s.pcRepository.FindByIdentifierAndOwner(ctx, pcIdentifier, ownerUserID)
	if err != nil {
		fmt.Printf("DEBUG RegisterPC: Error checking existing PC: %v\n", err)
		return nil, fmt.Errorf("error checking existing PC: %w", err)
	}

	fmt.Printf("DEBUG RegisterPC: Existing PC check completed, found: %v\n", existingPC != nil)

	// If PC already exists, update its status and last seen
	if existingPC != nil {
		fmt.Printf("DEBUG RegisterPC: Updating existing PC: %s\n", existingPC.PCID)
		existingPC.SetOnline()
		// Update IP if it has changed
		if existingPC.IP != ip {
			// Note: We might need to add an UpdateIP method to the domain entity
			// For now, we'll keep the existing IP
		}

		err = s.pcRepository.Save(ctx, existingPC)
		if err != nil {
			fmt.Printf("DEBUG RegisterPC: Error saving existing PC: %v\n", err)
			return nil, fmt.Errorf("error updating existing PC: %w", err)
		}

		fmt.Printf("DEBUG RegisterPC: Existing PC updated successfully: %s\n", existingPC.PCID)
		return existingPC, nil
	}

	fmt.Printf("DEBUG RegisterPC: Creating new PC\n")

	// Create new PC if it doesn't exist
	newPC, err := s.pcFactory.CreateClientPC(pcIdentifier, ip, ownerUserID)
	if err != nil {
		fmt.Printf("DEBUG RegisterPC: Error creating new PC: %v\n", err)
		return nil, fmt.Errorf("error creating new PC: %w", err)
	}

	fmt.Printf("DEBUG RegisterPC: New PC created with ID: %s\n", newPC.PCID)

	// Mark PC as online since it's being registered
	newPC.SetOnline()
	fmt.Printf("DEBUG RegisterPC: PC marked as online\n")

	// Save the new PC
	fmt.Printf("DEBUG RegisterPC: About to save new PC to repository\n")
	err = s.pcRepository.Save(ctx, newPC)
	if err != nil {
		fmt.Printf("DEBUG RegisterPC: Error saving new PC: %v\n", err)
		return nil, fmt.Errorf("error saving new PC: %w", err)
	}

	fmt.Printf("DEBUG RegisterPC: New PC saved successfully: %s\n", newPC.PCID)
	return newPC, nil
}

// GetPCByID retrieves a PC by its ID
func (s *PCService) GetPCByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error) {
	if pcID == "" {
		return nil, errors.New("PC ID cannot be empty")
	}

	pc, err := s.pcRepository.FindByID(ctx, pcID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving PC by ID: %w", err)
	}

	if pc == nil {
		return nil, errors.New("PC not found")
	}

	return pc, nil
}

// GetPCsByOwner retrieves all PCs belonging to a specific owner
func (s *PCService) GetPCsByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error) {
	if ownerUserID == "" {
		return nil, errors.New("owner user ID cannot be empty")
	}

	pcs, err := s.pcRepository.FindByOwner(ctx, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving PCs by owner: %w", err)
	}

	return pcs, nil
}

// GetOnlinePCsByOwner retrieves all online PCs belonging to a specific owner
func (s *PCService) GetOnlinePCsByOwner(ctx context.Context, ownerUserID string) ([]*clientpc.ClientPC, error) {
	if ownerUserID == "" {
		return nil, errors.New("owner user ID cannot be empty")
	}

	pcs, err := s.pcRepository.FindOnlineByOwner(ctx, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving online PCs by owner: %w", err)
	}

	return pcs, nil
}

// UpdatePCConnectionStatus updates the connection status of a PC
func (s *PCService) UpdatePCConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error {
	if pcID == "" {
		return errors.New("PC ID cannot be empty")
	}

	if !status.IsValid() {
		return errors.New("invalid connection status")
	}

	err := s.pcRepository.UpdateConnectionStatus(ctx, pcID, status)
	if err != nil {
		return fmt.Errorf("error updating PC connection status: %w", err)
	}

	return nil
}

// UpdatePCLastSeen updates the last seen timestamp of a PC
func (s *PCService) UpdatePCLastSeen(ctx context.Context, pcID string) error {
	if pcID == "" {
		return errors.New("PC ID cannot be empty")
	}

	err := s.pcRepository.UpdateLastSeen(ctx, pcID)
	if err != nil {
		return fmt.Errorf("error updating PC last seen: %w", err)
	}

	return nil
}

// GetAllClientPCs retrieves all client PCs in the system (for admin dashboard)
func (s *PCService) GetAllClientPCs(ctx context.Context) ([]*clientpc.ClientPC, error) {
	pcs, err := s.pcRepository.FindAll(ctx, 0, 0) // 0 means no limit
	if err != nil {
		return nil, fmt.Errorf("error retrieving all client PCs: %w", err)
	}

	return pcs, nil
}

// GetOnlineClientPCs retrieves all currently online client PCs (for admin dashboard)
func (s *PCService) GetOnlineClientPCs(ctx context.Context) ([]*clientpc.ClientPC, error) {
	// Primero obtenemos todos los PCs
	allPCs, err := s.GetAllClientPCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving PCs: %w", err)
	}

	// Filtrar solo los que están online
	onlinePCs := make([]*clientpc.ClientPC, 0) // Inicializar slice vacío en lugar de nil
	for _, pc := range allPCs {
		if pc.IsOnline() {
			onlinePCs = append(onlinePCs, pc)
		}
	}

	return onlinePCs, nil
}
