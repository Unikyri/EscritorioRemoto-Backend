package valueobjects

import (
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/shared/valueobjects"
)

// PCID es un Value Object específico para identificadores de PC
type PCID struct {
	*valueobjects.BaseID
}

// NewPCID crea un nuevo PCID validado
func NewPCID(value string) (*PCID, error) {
	baseID, err := valueobjects.NewBaseID(value)
	if err != nil {
		return nil, err
	}

	return &PCID{BaseID: baseID}, nil
}

// NewPCIDFromUUID crea un PCID desde un nuevo UUID
func NewPCIDFromUUID() *PCID {
	baseID := valueobjects.NewBaseIDFromUUID()
	return &PCID{BaseID: baseID}
}

// Equals compara dos PCIDs
func (id *PCID) Equals(other *PCID) bool {
	if other == nil {
		return false
	}
	return id.BaseID.Equals(other.BaseID)
}
