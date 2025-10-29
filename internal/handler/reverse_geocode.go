package handler

import (
	"context"
	"net/http"
	"strconv"

	"geocoding-api/internal/models"

	"github.com/gin-gonic/gin"
)

// ReverseGeocodeHandler handles reverse geocoding requests
type ReverseGeocodeHandler struct {
	service GeoCodingService
}

// Service interface for dependency injection
type GeoCodingService interface {
	ReverseGeocode(context.Context, float64, float64) (*models.Location, error)
}

// NewReverseGeocodeHandler creates a new reverse geocode handler
func NewReverseGeocodeHandler(svc GeoCodingService) *ReverseGeocodeHandler {
	return &ReverseGeocodeHandler{service: svc}
}

// ReverseGeocode handles GET /reverse-geocode requests
func (h *ReverseGeocodeHandler) ReverseGeocode(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	if latStr == "" || lonStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameters 'lat' and 'lon'"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid latitude format"})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid longitude format"})
		return
	}

	location, err := h.service.ReverseGeocode(c.Request.Context(), lat, lon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if location == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no address found near the specified coordinates"})
		return
	}

	c.JSON(http.StatusOK, location)
}
