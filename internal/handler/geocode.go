package handler

import (
	"context"
	"net/http"

	"geocoding-api/internal/models"

	"github.com/gin-gonic/gin"
)

// GeocodeHandler handles geocoding requests
type GeoCodeHandler struct {
	service GeoCodeService
}

// Service interface for dependency injection
type GeoCodeService interface {
	Geocode(context.Context, string) ([]models.Location, error)
}

// NewGeocodeHandler creates a new geocode handler
func NewGeoCodeHandler(svc GeoCodeService) *GeoCodeHandler {
	return &GeoCodeHandler{service: svc}
}

// Geocode handles GET /geocode requests
func (h *GeoCodeHandler) GeoCode(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameter 'q'"})
		return
	}

	locations, err := h.service.Geocode(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, locations)
}
