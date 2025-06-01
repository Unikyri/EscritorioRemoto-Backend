package pcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
)

// Mock implementations
type MockClientPCRepository struct {
	mock.Mock
}

func (m *MockClientPCRepository) Save(ctx context.Context, pc *clientpc.ClientPC) error {
	args := m.Called(ctx, pc)
	return args.Error(0)
}

func (m *MockClientPCRepository) FindByID(ctx context.Context, pcID string) (*clientpc.ClientPC, error) {
	args := m.Called(ctx, pcID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientpc.ClientPC), args.Error(1)
}

func (m *MockClientPCRepository) FindByIdentifierAndOwner(ctx context.Context, identifier string, ownerID string) (*clientpc.ClientPC, error) {
	args := m.Called(ctx, identifier, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientpc.ClientPC), args.Error(1)
}

func (m *MockClientPCRepository) FindByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]*clientpc.ClientPC), args.Error(1)
}

func (m *MockClientPCRepository) FindOnlineByOwner(ctx context.Context, ownerID string) ([]*clientpc.ClientPC, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]*clientpc.ClientPC), args.Error(1)
}

func (m *MockClientPCRepository) UpdateConnectionStatus(ctx context.Context, pcID string, status clientpc.PCConnectionStatus) error {
	args := m.Called(ctx, pcID, status)
	return args.Error(0)
}

func (m *MockClientPCRepository) UpdateLastSeen(ctx context.Context, pcID string) error {
	args := m.Called(ctx, pcID)
	return args.Error(0)
}

func (m *MockClientPCRepository) Delete(ctx context.Context, pcID string) error {
	args := m.Called(ctx, pcID)
	return args.Error(0)
}

func (m *MockClientPCRepository) FindAll(ctx context.Context, limit, offset int) ([]*clientpc.ClientPC, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*clientpc.ClientPC), args.Error(1)
}

func (m *MockClientPCRepository) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	args := m.Called(ctx, ownerID)
	return args.Int(0), args.Error(1)
}

type MockClientPCFactory struct {
	mock.Mock
}

func (m *MockClientPCFactory) CreateClientPC(identifier, ip, ownerUserID string) (*clientpc.ClientPC, error) {
	args := m.Called(identifier, ip, ownerUserID)
	return args.Get(0).(*clientpc.ClientPC), args.Error(1)
}

// Test cases
func TestPCService_RegisterPC_NewPC(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	ownerUserID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID
	pcIdentifier := "test-pc"
	ip := "192.168.1.100"

	// Mock: PC doesn't exist
	mockRepo.On("FindByIdentifierAndOwner", ctx, pcIdentifier, ownerUserID).Return(nil, nil)

	// Mock: Factory creates new PC
	newPC, _ := clientpc.NewClientPC("550e8400-e29b-41d4-a716-446655440001", pcIdentifier, ip, ownerUserID)
	mockFactory.On("CreateClientPC", pcIdentifier, ip, ownerUserID).Return(newPC, nil)

	// Mock: Save succeeds
	mockRepo.On("Save", ctx, mock.AnythingOfType("*clientpc.ClientPC")).Return(nil)

	// Act
	result, err := service.RegisterPC(ctx, ownerUserID, pcIdentifier, ip)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, pcIdentifier, result.Identifier)
	assert.Equal(t, ip, result.IP)
	assert.Equal(t, ownerUserID, result.OwnerUserID)
	assert.True(t, result.IsOnline())

	mockRepo.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
}

func TestPCService_RegisterPC_ExistingPC(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	ownerUserID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID
	pcIdentifier := "test-pc"
	ip := "192.168.1.100"

	// Mock: PC already exists
	existingPC, _ := clientpc.NewClientPC("550e8400-e29b-41d4-a716-446655440001", pcIdentifier, "192.168.1.50", ownerUserID)
	existingPC.SetOffline()
	mockRepo.On("FindByIdentifierAndOwner", ctx, pcIdentifier, ownerUserID).Return(existingPC, nil)

	// Mock: Save succeeds
	mockRepo.On("Save", ctx, mock.AnythingOfType("*clientpc.ClientPC")).Return(nil)

	// Act
	result, err := service.RegisterPC(ctx, ownerUserID, pcIdentifier, ip)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, pcIdentifier, result.Identifier)
	assert.Equal(t, ownerUserID, result.OwnerUserID)
	assert.True(t, result.IsOnline())

	mockRepo.AssertExpectations(t)
	// Factory should not be called for existing PC
	mockFactory.AssertNotCalled(t, "CreateClientPC")
}

func TestPCService_RegisterPC_InvalidInput(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()

	// Test cases for invalid input
	testCases := []struct {
		name         string
		ownerUserID  string
		pcIdentifier string
		ip           string
		expectedErr  string
	}{
		{"Empty owner ID", "", "test-pc", "192.168.1.100", "owner user ID cannot be empty"},
		{"Empty PC identifier", "550e8400-e29b-41d4-a716-446655440000", "", "192.168.1.100", "PC identifier cannot be empty"},
		{"Empty IP", "550e8400-e29b-41d4-a716-446655440000", "test-pc", "", "IP address cannot be empty"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result, err := service.RegisterPC(ctx, tc.ownerUserID, tc.pcIdentifier, tc.ip)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestPCService_RegisterPC_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	ownerUserID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID
	pcIdentifier := "test-pc"
	ip := "192.168.1.100"

	// Mock: Repository error
	mockRepo.On("FindByIdentifierAndOwner", ctx, pcIdentifier, ownerUserID).Return(nil, errors.New("database error"))

	// Act
	result, err := service.RegisterPC(ctx, ownerUserID, pcIdentifier, ip)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "error checking existing PC")

	mockRepo.AssertExpectations(t)
}

func TestPCService_GetPCByID_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	pcID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID

	// Mock: PC found
	expectedPC, _ := clientpc.NewClientPC(pcID, "test-pc", "192.168.1.100", "550e8400-e29b-41d4-a716-446655440000")
	mockRepo.On("FindByID", ctx, pcID).Return(expectedPC, nil)

	// Act
	result, err := service.GetPCByID(ctx, pcID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, pcID, result.PCID)

	mockRepo.AssertExpectations(t)
}

func TestPCService_GetPCByID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	pcID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID

	// Mock: PC not found
	mockRepo.On("FindByID", ctx, pcID).Return((*clientpc.ClientPC)(nil), nil)

	// Act
	result, err := service.GetPCByID(ctx, pcID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "PC not found")

	mockRepo.AssertExpectations(t)
}

func TestPCService_UpdatePCConnectionStatus_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	pcID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID
	status := clientpc.PCConnectionStatusOnline

	// Mock: Update succeeds
	mockRepo.On("UpdateConnectionStatus", ctx, pcID, status).Return(nil)

	// Act
	err := service.UpdatePCConnectionStatus(ctx, pcID, status)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestPCService_UpdatePCConnectionStatus_InvalidStatus(t *testing.T) {
	// Arrange
	mockRepo := new(MockClientPCRepository)
	mockFactory := new(MockClientPCFactory)
	service := NewPCService(mockRepo, mockFactory)

	ctx := context.Background()
	pcID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID
	status := clientpc.PCConnectionStatus("INVALID")

	// Act
	err := service.UpdatePCConnectionStatus(ctx, pcID, status)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid connection status")

	// Repository should not be called
	mockRepo.AssertNotCalled(t, "UpdateConnectionStatus")
}
