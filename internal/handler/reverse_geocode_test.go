package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"geocoding-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReverseGeoCodeService is a mock implementation of the ReverseGeoCodeService interface
type MockReverseGeoCodeService struct {
	mock.Mock
}

func (m *MockReverseGeoCodeService) ReverseGeocode(ctx context.Context, lat float64, lon float64) (*models.Location, error) {
	args := m.Called(ctx, lat, lon)
	return args.Get(0).(*models.Location), args.Error(1)
}

func TestReverseGeoCodeHandler_ReverseGeocode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		lat            float64
		lon            float64
		mockLocation   *models.Location
		mockError      error
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "missing query parameter",
			lat:            0,
			lon:            0,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "missing required query parameter 'q'"},
		},
		{
			name: "successful geocoding with results",
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
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: models.Location{
				ID:           1,
				Prefecture:   "東京都",
				Municipality: "千代田区",
				Address1:     "丸の内",
				Latitude:     35.681236,
				Longitude:    139.767125,
			},
		},
		{
			name:           "successful geocoding with no results",
			lat:            35.681236,
			lon:            139.767125,
			mockLocation:   nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
			expectedBody:   gin.H{"error": "no address found near the specified coordinates"},
		},
		{
			name:           "service error",
			lat:            35.681236,
			lon:            139.767125,
			mockLocation:   nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   gin.H{"error": "internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockSvc := new(MockReverseGeoCodeService)
			handler := NewReverseGeocodeHandler(mockSvc)

			if tt.lat != 0 && tt.lon != 0 {
				mockSvc.On("ReverseGeocode", mock.Anything, tt.lat, tt.lon).Return(tt.mockLocation, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/reverse-geocode", nil)
			if tt.lat != 0 && tt.lon != 0 {
				q := req.URL.Query()
				q.Add("lat", strconv.FormatFloat(tt.lat, 'f', -1, 64))
				q.Add("lon", strconv.FormatFloat(tt.lon, 'f', -1, 64))
				req.URL.RawQuery = q.Encode()
			}
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.ReverseGeocode(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var actualBody interface{}
			err := json.Unmarshal(w.Body.Bytes(), &actualBody)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, actualBody)

			if tt.lat != 0 && tt.lon != 0 {
				mockSvc.AssertExpectations(t)
			}
		})
	}
}
