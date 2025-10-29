package repository

import (
	"context"
	"fmt"

	"geocoding-api/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implements the repository interface for PostgreSQL
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SearchLocationsByText performs a full-text search on the locations table
func (r *Repository) SearchLocationsByText(ctx context.Context, query string) ([]models.Location, error) {
	sql := `
		SELECT
			id,
			prefecture,
			municipality,
			address_1,
			address_2,
			block_lot,
			ST_Y(geom) as latitude,
			ST_X(geom) as longitude
		FROM locations
		WHERE full_address_tsvector @@ to_tsquery('japanese', $1)
		ORDER BY ts_rank(full_address_tsvector, to_tsquery('japanese', $1)) DESC
		LIMIT 10
	`

	rows, err := r.db.Query(ctx, sql, query)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to execute search query: %w", err)
	}
	defer rows.Close()

	var locations []models.Location
	for rows.Next() {
		var loc models.Location
		err := rows.Scan(
			&loc.ID,
			&loc.Prefecture,
			&loc.Municipality,
			&loc.Address1,
			&loc.Address2,
			&loc.BlockLot,
			&loc.Latitude,
			&loc.Longitude,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: failed to scan location: %w", err)
		}
		locations = append(locations, loc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating rows: %w", err)
	}

	return locations, nil
}

// FindNearestLocation performs a spatial query to find the nearest location to the given coordinates
func (r *Repository) FindNearestLocation(ctx context.Context, lat, lon float64) (*models.Location, error) {
	sql := `
		SELECT
			id,
			prefecture,
			municipality,
			address_1,
			address_2,
			block_lot,
			ST_Y(geom) as latitude,
			ST_X(geom) as longitude
		FROM locations
		WHERE ST_DWithin(geom, ST_SetSRID(ST_MakePoint($2, $1), 4326), 10000) -- Within 10km
		ORDER BY geom <-> ST_SetSRID(ST_MakePoint($2, $1), 4326)
		LIMIT 1
	`

	var loc models.Location
	err := r.db.QueryRow(ctx, sql, lat, lon).Scan(
		&loc.ID,
		&loc.Prefecture,
		&loc.Municipality,
		&loc.Address1,
		&loc.Address2,
		&loc.BlockLot,
		&loc.Latitude,
		&loc.Longitude,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("repository: no location found near coordinates")
		}
		return nil, fmt.Errorf("repository: failed to execute spatial query: %w", err)
	}

	return &loc, nil
}