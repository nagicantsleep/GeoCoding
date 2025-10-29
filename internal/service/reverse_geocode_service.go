package service

import (
	"context"
	"fmt"

	"geocoding-api/internal/models"
)

// ReverseGeoCodeService contains the core business logic for reverse geocoding operations
type ReverseGeoCodeService struct {
	repo ReverseGeoCodeRepository
}

// ReverseGeoCodeRepository interface for dependency injection
type ReverseGeoCodeRepository interface {
	FindNearestLocation(ctx context.Context, lat, lon float64) (*models.Location, error)
}

// NewReverseGeoCodeService creates a new reverse geo code service
func NewReverseGeoCodeService(repo ReverseGeoCodeRepository) *ReverseGeoCodeService {
	return &ReverseGeoCodeService{repo: repo}
}

// ReverseGeocode finds the nearest address to the given coordinates using spatial query
func (s *ReverseGeoCodeService) ReverseGeocode(ctx context.Context, lat, lon float64) (*models.Location, error) {
	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("service: invalid latitude: %f", lat)
	}
	if lon < -180 || lon > 180 {
		return nil, fmt.Errorf("service: invalid longitude: %f", lon)
	}

	location, err := s.repo.FindNearestLocation(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("service: failed to find nearest location: %w", err)
	}

	return location, nil
}
