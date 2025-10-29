package service

import (
	"context"
	"testing"

	"geocoding-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockGeoCodeRepository struct {
	mock.Mock
}

// SearchLocationsByText implements GeoCodeRepository.
func (m *MockGeoCodeRepository) SearchLocationsByText(ctx context.Context, query string) ([]models.Location, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]models.Location), args.Error(1)
}

func TestGeoCodeService_Geocode(t *testing.T) {
	tests := []struct {
		name          string
		address       string
		mockLocations []models.Location
		mockError     error
		expected      []models.Location
		expectError   bool
	}{
		{
			name:        "empty address",
			address:     "",
			expectError: true,
		},
		{
			name:    "successful search with results",
			address: "東京都千代田区丸の内",
			mockLocations: []models.Location{
				{
					ID:           1,
					Prefecture:   "東京都",
					Municipality: "千代田区",
					Address1:     "丸の内",
					Latitude:     35.681236,
					Longitude:    139.767125,
				},
			},
			mockError: nil,
			expected: []models.Location{
				{
					ID:           1,
					Prefecture:   "東京都",
					Municipality: "千代田区",
					Address1:     "丸の内",
					Latitude:     35.681236,
					Longitude:    139.767125,
				},
			},
			expectError: false,
		},
		{
			name:          "successful search with no results",
			address:       "nonexistent address",
			mockLocations: []models.Location{},
			mockError:     nil,
			expected:      []models.Location{},
			expectError:   false,
		},
		{
			name:        "repository error",
			address:     "東京都千代田区丸の内",
			mockError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockGeoCodeRepository)
			service := NewGeoCodeService(mockRepo)

			if tt.address != "" {
				mockRepo.On("SearchLocationsByText", mock.Anything, tt.address).Return(tt.mockLocations, tt.mockError)
			}

			// Execute
			result, err := service.Geocode(context.Background(), tt.address)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			if tt.address != "" {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}
