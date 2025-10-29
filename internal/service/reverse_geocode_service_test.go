package service

import (
	"context"
	"testing"

	"geocoding-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockReverseGeoCodeRepository struct {
	mock.Mock
}

// FindNearestLocation implements ReverseGeoCodeRepository.
func (m *MockReverseGeoCodeRepository) FindNearestLocation(ctx context.Context, lat float64, lon float64) (*models.Location, error) {
	args := m.Called(ctx, lat, lon)
	return args.Get(0).(*models.Location), args.Error(1)
}

func TestReverseGeoCodeService_ReverseGeocode(t *testing.T) {
	tests := []struct {
		name          string
		lat           float64
		lon           float64
		mockLocation  *models.Location
		mockError     error
		expected      *models.Location
		expectError   bool
	}{
		{
			name:        "empty lat and lon",
			lat:         0,
			lon:         0,
			expectError: true,
		},
		{
			name: "successful search with results",
			lat:  35.681236,
			lon:  139.767125,
			mockLocation: &models.Location{
				ID:           1,
				Prefecture:   "東京都",
				Municipality: "千代田区",
				Address1:     "丸の内",
				Latitude:     35.681236,
				Longitude:    139.767125,
			},
			mockError: nil,
			expected: &models.Location{
				ID:           1,
				Prefecture:   "東京都",
				Municipality: "千代田区",
				Address1:     "丸の内",
				Latitude:     35.681236,
				Longitude:    139.767125,
			},
			expectError: false,
		},
		{
			name:          "successful search with no results",
			lat:           35.681236,
			lon:           139.767125,
			mockLocation:  nil,
			mockError:     nil,
			expected:      nil,
			expectError:   false,
		},
		{
			name:        "repository error",
			lat:         35.681236,
			lon:         139.767125,
			mockError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockReverseGeoCodeRepository)
			service := NewReverseGeoCodeService(mockRepo)

			if tt.lat != 0 && tt.lon != 0 {
				mockRepo.On("FindNearestLocation", mock.Anything, tt.lat, tt.lon).Return(tt.mockLocation, tt.mockError)
			}

			// Execute
			result, err := service.ReverseGeocode(context.Background(), tt.lat, tt.lon)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			if tt.lat != 0 && tt.lon != 0 {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}
