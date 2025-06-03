package services

import (
	"errors"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc/entities"
)

// IPCDomainService define la interface del servicio de dominio
type IPCDomainService interface {
	// RegisterOrUpdatePC maneja el registro o actualización de un PC
	RegisterOrUpdatePC(existingPC *entities.ClientPC, identifier, ip, ownerUserID string) (*entities.ClientPC, error)

	// CanPCGoOnline verifica si un PC puede pasar al estado online
	CanPCGoOnline(pc *entities.ClientPC) error

	// HandlePCConnectionTimeout maneja el timeout de conexión de un PC
	HandlePCConnectionTimeout(pc *entities.ClientPC, timeout time.Duration) error

	// ValidatePCIdentifier valida que un identificador de PC sea válido
	ValidatePCIdentifier(identifier string) error

	// ValidateIPAddress valida que una dirección IP sea válida
	ValidateIPAddress(ip string) error

	// ShouldCreateNewPC determina si se debe crear un nuevo PC o actualizar uno existente
	ShouldCreateNewPC(existingPC *entities.ClientPC, identifier, ownerUserID string) bool
}

// PCDomainService implementa la lógica de negocio para PCs
type PCDomainService struct{}

// NewPCDomainService crea una nueva instancia del servicio de dominio
func NewPCDomainService() IPCDomainService {
	return &PCDomainService{}
}

// RegisterOrUpdatePC maneja el registro o actualización de un PC
func (s *PCDomainService) RegisterOrUpdatePC(existingPC *entities.ClientPC, identifier, ip, ownerUserID string) (*entities.ClientPC, error) {
	// Validar parámetros de entrada
	if err := s.ValidatePCIdentifier(identifier); err != nil {
		return nil, err
	}

	if err := s.ValidateIPAddress(ip); err != nil {
		return nil, err
	}

	if ownerUserID == "" {
		return nil, errors.New("owner user ID cannot be empty")
	}

	// Si no existe PC, crear uno nuevo
	if existingPC == nil {
		newPC, err := entities.NewClientPC(identifier, ip, ownerUserID)
		if err != nil {
			return nil, err
		}

		// El PC se crea offline, pero se puede poner online inmediatamente si es válido
		if err := s.CanPCGoOnline(newPC); err == nil {
			if err := newPC.SetOnline(); err != nil {
				return nil, err
			}
		}

		return newPC, nil
	}

	// Actualizar PC existente
	if existingPC.OwnerUserID() != ownerUserID {
		return nil, errors.New("PC identifier already registered by another user")
	}

	// Actualizar IP si es diferente
	if existingPC.IP() != ip {
		if err := existingPC.UpdateIP(ip); err != nil {
			return nil, err
		}
	}

	// Marcar como online si puede
	if err := s.CanPCGoOnline(existingPC); err == nil {
		if err := existingPC.SetOnline(); err != nil {
			return nil, err
		}
	}

	// Actualizar último visto
	existingPC.UpdateLastSeen()

	return existingPC, nil
}

// CanPCGoOnline verifica si un PC puede pasar al estado online
func (s *PCDomainService) CanPCGoOnline(pc *entities.ClientPC) error {
	if pc == nil {
		return errors.New("PC cannot be nil")
	}

	// Validar que el PC tenga datos válidos
	if pc.Identifier() == "" {
		return errors.New("PC must have a valid identifier")
	}

	if pc.IP() == "" {
		return errors.New("PC must have a valid IP address")
	}

	if pc.OwnerUserID() == "" {
		return errors.New("PC must have a valid owner")
	}

	// El PC puede ir online desde cualquier estado
	return nil
}

// HandlePCConnectionTimeout maneja el timeout de conexión de un PC
func (s *PCDomainService) HandlePCConnectionTimeout(pc *entities.ClientPC, timeout time.Duration) error {
	if pc == nil {
		return errors.New("PC cannot be nil")
	}

	// Solo manejar timeout si el PC está online
	if !pc.IsOnline() {
		return nil // No hay nada que hacer si ya está offline
	}

	// Verificar si la conexión ha expirado
	if pc.IsConnectionExpired(timeout) {
		reason := "connection timeout"
		return pc.SetOffline(reason)
	}

	return nil
}

// ValidatePCIdentifier valida que un identificador de PC sea válido
func (s *PCDomainService) ValidatePCIdentifier(identifier string) error {
	if identifier == "" {
		return errors.New("PC identifier cannot be empty")
	}

	// Validaciones de formato
	if len(identifier) < 3 {
		return errors.New("PC identifier must be at least 3 characters long")
	}

	if len(identifier) > 50 {
		return errors.New("PC identifier cannot exceed 50 characters")
	}

	// Validar caracteres permitidos (alfanuméricos, guiones, guiones bajos)
	for _, char := range identifier {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return errors.New("PC identifier can only contain alphanumeric characters, hyphens, and underscores")
		}
	}

	return nil
}

// ValidateIPAddress valida que una dirección IP sea válida
func (s *PCDomainService) ValidateIPAddress(ip string) error {
	if ip == "" {
		return errors.New("IP address cannot be empty")
	}

	// Validación básica de formato IPv4
	// En una implementación real, se podría usar net.ParseIP
	if len(ip) < 7 || len(ip) > 15 {
		return errors.New("IP address format is invalid")
	}

	// Verificar que no sea localhost en producción
	if ip == "127.0.0.1" || ip == "localhost" {
		// En desarrollo está bien, pero podría ser una regla de negocio
		// return errors.New("localhost IP is not allowed")
	}

	return nil
}

// ShouldCreateNewPC determina si se debe crear un nuevo PC o actualizar uno existente
func (s *PCDomainService) ShouldCreateNewPC(existingPC *entities.ClientPC, identifier, ownerUserID string) bool {
	// Si no existe PC, crear uno nuevo
	if existingPC == nil {
		return true
	}

	// Si el PC existe pero tiene diferente owner, no se puede actualizar
	if existingPC.OwnerUserID() != ownerUserID {
		return false // Esto generará un error en RegisterOrUpdatePC
	}

	// Si el PC existe y tiene el mismo owner, actualizar
	return false
}
