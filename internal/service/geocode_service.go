package service

import (
	"context"
	"fmt"

	"geocoding-api/internal/models"
)

// GeocodeService contains the core business logic for geocoding operations
type GeoCodeService struct {
	repo GeoCodeRepository
}

// Repository interface for dependency injection
type GeoCodeRepository interface {
	SearchLocationsByText(ctx context.Context, query string) ([]models.Location, error)
}

// NewGeoCodeService creates a new geo code service
func NewGeoCodeService(repo GeoCodeRepository) *GeoCodeService {
	return &GeoCodeService{repo: repo}
}

// Geocode searches for locations by address text using full-text search
func (s *GeoCodeService) Geocode(ctx context.Context, address string) ([]models.Location, error) {
	if address == "" {
		return nil, fmt.Errorf("service: address cannot be empty")
	}

	locations, err := s.repo.SearchLocationsByText(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("service: failed to search locations: %w", err)
	}

	return locations, nil
}


