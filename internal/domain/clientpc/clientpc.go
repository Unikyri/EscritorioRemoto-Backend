package clientpc

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// PCConnectionStatus represents the connection status of a client PC
type PCConnectionStatus string

const (
	PCConnectionStatusOnline     PCConnectionStatus = "ONLINE"
	PCConnectionStatusOffline    PCConnectionStatus = "OFFLINE"
	PCConnectionStatusConnecting PCConnectionStatus = "CONNECTING"
)

// IsValid checks if the connection status is valid
func (status PCConnectionStatus) IsValid() bool {
	switch status {
	case PCConnectionStatusOnline, PCConnectionStatusOffline, PCConnectionStatusConnecting:
		return true
	default:
		return false
	}
}

// String returns the string representation of the connection status
func (status PCConnectionStatus) String() string {
	return string(status)
}

// ClientPC represents a client PC registered in the system
type ClientPC struct {
	PCID             string             `json:"pcId" db:"pc_id"`
	Identifier       string             `json:"identifier" db:"identifier"`
	IP               string             `json:"ip" db:"ip"`
	ConnectionStatus PCConnectionStatus `json:"connectionStatus" db:"connection_status"`
	RegisteredAt     time.Time          `json:"registeredAt" db:"registered_at"`
	OwnerUserID      string             `json:"ownerUserId" db:"owner_user_id"`
	LastSeenAt       *time.Time         `json:"lastSeenAt" db:"last_seen_at"`
	CreatedAt        time.Time          `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time          `json:"updatedAt" db:"updated_at"`
}

// NewClientPC creates a new ClientPC instance with validation
func NewClientPC(pcID, identifier, ip, ownerUserID string) (*ClientPC, error) {
	if err := validatePCID(pcID); err != nil {
		return nil, err
	}

	if err := validateIdentifier(identifier); err != nil {
		return nil, err
	}

	if err := validateIP(ip); err != nil {
		return nil, err
	}

	if err := validateOwnerUserID(ownerUserID); err != nil {
		return nil, err
	}

	now := time.Now()

	return &ClientPC{
		PCID:             pcID,
		Identifier:       identifier,
		IP:               ip,
		ConnectionStatus: PCConnectionStatusOffline,
		RegisteredAt:     now,
		OwnerUserID:      ownerUserID,
		LastSeenAt:       nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// SetOnline marks the PC as online and updates LastSeenAt
func (pc *ClientPC) SetOnline() {
	now := time.Now()
	pc.ConnectionStatus = PCConnectionStatusOnline
	pc.LastSeenAt = &now
	pc.UpdatedAt = now
}

// SetOffline marks the PC as offline
func (pc *ClientPC) SetOffline() {
	pc.ConnectionStatus = PCConnectionStatusOffline
	pc.UpdatedAt = time.Now()
}

// SetConnecting marks the PC as connecting
func (pc *ClientPC) SetConnecting() {
	pc.ConnectionStatus = PCConnectionStatusConnecting
	pc.UpdatedAt = time.Now()
}

// UpdateLastSeen updates the last seen timestamp
func (pc *ClientPC) UpdateLastSeen() {
	now := time.Now()
	pc.LastSeenAt = &now
	pc.UpdatedAt = now
}

// IsOnline returns true if the PC is currently online
func (pc *ClientPC) IsOnline() bool {
	return pc.ConnectionStatus == PCConnectionStatusOnline
}

// IsOffline returns true if the PC is currently offline
func (pc *ClientPC) IsOffline() bool {
	return pc.ConnectionStatus == PCConnectionStatusOffline
}

// IsConnecting returns true if the PC is currently connecting
func (pc *ClientPC) IsConnecting() bool {
	return pc.ConnectionStatus == PCConnectionStatusConnecting
}

// Validate validates all fields of the ClientPC
func (pc *ClientPC) Validate() error {
	if err := validatePCID(pc.PCID); err != nil {
		return err
	}

	if err := validateIdentifier(pc.Identifier); err != nil {
		return err
	}

	if err := validateIP(pc.IP); err != nil {
		return err
	}

	if err := validateOwnerUserID(pc.OwnerUserID); err != nil {
		return err
	}

	if !pc.ConnectionStatus.IsValid() {
		return errors.New("invalid connection status")
	}

	return nil
}

// Domain validation functions

func validatePCID(pcID string) error {
	if strings.TrimSpace(pcID) == "" {
		return errors.New("PC ID cannot be empty")
	}

	// UUID format validation
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(pcID) {
		return errors.New("PC ID must be a valid UUID format")
	}

	return nil
}

func validateIdentifier(identifier string) error {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return errors.New("PC identifier cannot be empty")
	}

	if len(identifier) < 3 {
		return errors.New("PC identifier must be at least 3 characters long")
	}

	if len(identifier) > 50 {
		return errors.New("PC identifier must be less than 50 characters long")
	}

	// Allow alphanumeric, hyphens, underscores
	validRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validRegex.MatchString(identifier) {
		return errors.New("PC identifier can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

func validateIP(ip string) error {
	if strings.TrimSpace(ip) == "" {
		return errors.New("IP address cannot be empty")
	}

	// Basic IP validation (IPv4)
	ipRegex := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	if !ipRegex.MatchString(ip) {
		return errors.New("invalid IP address format")
	}

	return nil
}

func validateOwnerUserID(ownerUserID string) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return errors.New("owner user ID cannot be empty")
	}

	// UUID format validation
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(ownerUserID) {
		return errors.New("owner user ID must be a valid UUID format")
	}

	return nil
}
