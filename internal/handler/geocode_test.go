package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"geocoding-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGeoCodeService is a mock implementation of the GeoCodeService interface
type MockGeoCodeService struct {
	mock.Mock
}

func (m *MockGeoCodeService) Geocode(ctx context.Context, address string) ([]models.Location, error) {
	args := m.Called(ctx, address)
	return args.Get(0).([]models.Location), args.Error(1)
}

func TestGeoCodeHandler_Geocode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		mockLocations  []models.Location
		mockError      error
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "missing query parameter",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   gin.H{"error": "missing required query parameter 'q'"},
		},
		{
			name:  "successful geocoding with results",
			query: "東京都千代田区丸の内",
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
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: []models.Location{
				{
					ID:           1,
					Prefecture:   "東京都",
					Municipality: "千代田区",
					Address1:     "丸の内",
					Latitude:     35.681236,
					Longitude:    139.767125,
				},
			},
		},
		{
			name:           "successful geocoding with no results",
			query:          "nonexistent address",
			mockLocations:  []models.Location{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   []models.Location{},
		},
		{
			name:           "service error",
			query:          "東京都千代田区丸の内",
			mockLocations:  nil,
			mockError:      assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   gin.H{"error": "internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockSvc := new(MockGeoCodeService)
			handler := NewGeoCodeHandler(mockSvc)

			if tt.query != "" {
				mockSvc.On("Geocode", mock.Anything, tt.query).Return(tt.mockLocations, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/geocode", nil)
			if tt.query != "" {
				q := req.URL.Query()
				q.Add("q", tt.query)
				req.URL.RawQuery = q.Encode()
			}
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.GeoCode(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var actualBody interface{}
			err := json.Unmarshal(w.Body.Bytes(), &actualBody)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, actualBody)

			if tt.query != "" {
				mockSvc.AssertExpectations(t)
			}
		})
	}
}
