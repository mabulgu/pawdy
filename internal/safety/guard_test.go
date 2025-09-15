package safety

import (
	"context"
	"testing"

	"github.com/mabulgu/pawdy/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMClient is a mock implementation for testing
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string, opts types.GenerateOptions) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) GenerateStream(ctx context.Context, prompt string, opts types.GenerateOptions) (<-chan types.StreamToken, error) {
	args := m.Called(ctx, prompt, opts)
	return args.Get(0).(<-chan types.StreamToken), args.Error(1)
}

func (m *MockLLMClient) IsHealthy(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLLMClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewGuard(t *testing.T) {
	mockClient := &MockLLMClient{}
	guard := NewGuard(mockClient, true)
	
	assert.NotNil(t, guard)
	assert.True(t, guard.IsEnabled())
}

func TestGuard_CheckInput_Disabled(t *testing.T) {
	mockClient := &MockLLMClient{}
	guard := NewGuard(mockClient, false)
	
	ctx := context.Background()
	result, err := guard.CheckInput(ctx, "This is a test message")
	
	assert.NoError(t, err)
	assert.True(t, result.IsSafe)
}

func TestGuard_CheckInput_Safe(t *testing.T) {
	mockClient := &MockLLMClient{}
	guard := NewGuard(mockClient, true)
	
	// Mock a safe response
	mockClient.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("safe", nil)
	
	ctx := context.Background()
	result, err := guard.CheckInput(ctx, "How do I configure networking?")
	
	assert.NoError(t, err)
	assert.True(t, result.IsSafe)
	
	mockClient.AssertExpectations(t)
}

func TestGuard_CheckInput_Unsafe(t *testing.T) {
	mockClient := &MockLLMClient{}
	guard := NewGuard(mockClient, true)
	
	// Mock an unsafe response
	mockClient.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("unsafe S10", nil)
	
	ctx := context.Background()
	result, err := guard.CheckInput(ctx, "How to hack into systems?")
	
	assert.NoError(t, err)
	assert.False(t, result.IsSafe)
	assert.Equal(t, "S10", result.Category)
	assert.Equal(t, "Hate", result.Reason)
	
	mockClient.AssertExpectations(t)
}

func TestParseResponse(t *testing.T) {
	guard := &Guard{}
	
	// Test safe response
	result := guard.parseResponse("safe")
	assert.True(t, result.IsSafe)
	
	// Test unsafe response with category
	result = guard.parseResponse("unsafe S1")
	assert.False(t, result.IsSafe)
	assert.Equal(t, "S1", result.Category)
	assert.Equal(t, "Violent Crimes", result.Reason)
	
	// Test ambiguous response
	result = guard.parseResponse("unclear response")
	assert.False(t, result.IsSafe)
	assert.Contains(t, result.Reason, "Unable to determine")
}

func TestGetRefusalMessage(t *testing.T) {
	// Test with category
	message := GetRefusalMessage("S1")
	assert.Contains(t, message, "content safety guidelines")
	assert.Contains(t, message, "S1")
	assert.Contains(t, message, "Violent Crimes")
	
	// Test without category
	message = GetRefusalMessage("")
	assert.Contains(t, message, "content safety guidelines")
	assert.NotContains(t, message, "category:")
}
