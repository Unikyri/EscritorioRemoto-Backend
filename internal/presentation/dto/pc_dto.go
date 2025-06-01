package dto

import "time"

// ClientPCDTO represents a client PC for API responses
type ClientPCDTO struct {
	PCID             string     `json:"pcId"`
	Identifier       string     `json:"identifier"`
	ConnectionStatus string     `json:"connectionStatus"`
	OwnerUsername    string     `json:"ownerUsername"`
	IP               string     `json:"ip"`
	RegisteredAt     time.Time  `json:"registeredAt"`
	LastSeenAt       *time.Time `json:"lastSeenAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// ClientPCListResponse represents the response for getting all client PCs
type ClientPCListResponse struct {
	Success bool          `json:"success"`
	Data    []ClientPCDTO `json:"data"`
	Count   int           `json:"count"`
	Message string        `json:"message,omitempty"`
}

// OnlineClientPCListResponse represents the response for getting online client PCs
type OnlineClientPCListResponse struct {
	Success bool          `json:"success"`
	Data    []ClientPCDTO `json:"data"`
	Count   int           `json:"count"`
	Message string        `json:"message,omitempty"`
}
