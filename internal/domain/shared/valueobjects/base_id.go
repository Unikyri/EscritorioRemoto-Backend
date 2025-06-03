package valueobjects

import (
	"errors"
	"regexp"

	"github.com/google/uuid"
)

// BaseID es un Value Object base para todos los identificadores del sistema
type BaseID struct {
	value string
}

// NewBaseID crea un nuevo BaseID validado
func NewBaseID(value string) (*BaseID, error) {
	if err := validateID(value); err != nil {
		return nil, err
	}

	return &BaseID{value: value}, nil
}

// NewBaseIDFromUUID crea un BaseID desde un UUID
func NewBaseIDFromUUID() *BaseID {
	return &BaseID{value: uuid.New().String()}
}

// Value retorna el valor del ID
func (id *BaseID) Value() string {
	return id.value
}

// String implementa fmt.Stringer
func (id *BaseID) String() string {
	return id.value
}

// Equals compara dos BaseIDs
func (id *BaseID) Equals(other *BaseID) bool {
	if other == nil {
		return false
	}
	return id.value == other.value
}

// IsEmpty verifica si el ID está vacío
func (id *BaseID) IsEmpty() bool {
	return id.value == ""
}

// validateID valida que el ID sea un UUID válido
func validateID(value string) error {
	if value == "" {
		return errors.New("ID cannot be empty")
	}

	// Validar formato UUID
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(value) {
		return errors.New("ID must be a valid UUID format")
	}

	return nil
}
