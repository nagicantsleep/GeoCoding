//go:build integration

package repository

import (
	"context"
	"testing"

	"geocoding-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	// Start PostgreSQL container with PostGIS
	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		postgresC.Terminate(ctx)
	})

	host, err := postgresC.Host(ctx)
	require.NoError(t, err)

	port, err := postgresC.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connString := "postgres://testuser:testpass@" + host + ":" + port.Port() + "/testdb?sslmode=disable"

	// Connect to database
	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	// Create test schema
	_, err = pool.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS postgis;

		CREATE TABLE locations (
			id BIGSERIAL PRIMARY KEY,
			prefecture VARCHAR(255),
			municipality VARCHAR(255),
			address_1 VARCHAR(255),
			address_2 VARCHAR(255),
			block_lot VARCHAR(255),
			full_address_tsvector TSVECTOR GENERATED ALWAYS AS (
				to_tsvector('japanese', prefecture || municipality || address_1 || address_2)
			) STORED,
			geom GEOGRAPHY(POINT, 4326)
		);

		CREATE INDEX locations_geom_idx ON locations USING GIST (geom);
		CREATE INDEX locations_full_address_tsvector_idx ON locations USING GIN (full_address_tsvector);

		-- Insert test data
		INSERT INTO locations (prefecture, municipality, address_1, address_2, geom) VALUES
		('東京都', '千代田区', '丸の内', '', ST_SetSRID(ST_MakePoint(139.767125, 35.681236), 4326)),
		('東京都', '港区', '赤坂', '1丁目', ST_SetSRID(ST_MakePoint(139.732, 35.675), 4326));
	`)
	require.NoError(t, err)

	return pool
}

func TestPostgresRepository_SearchLocationsByText(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDatabase(t)
	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	tests := []struct {
		name     string
		query    string
		expected []models.Location
	}{
		{
			name:  "search by municipality",
			query: "千代田区",
			expected: []models.Location{
				{
					ID:          1,
					Prefecture:  "東京都",
					Municipality: "千代田区",
					Address1:    "丸の内",
					Address2:    "",
					Latitude:    35.681236,
					Longitude:   139.767125,
				},
			},
		},
		{
			name:  "search by address",
			query: "丸の内",
			expected: []models.Location{
				{
					ID:          1,
					Prefecture:  "東京都",
					Municipality: "千代田区",
					Address1:    "丸の内",
					Address2:    "",
					Latitude:    35.681236,
					Longitude:   139.767125,
				},
			},
		},
		{
			name:     "search with no results",
			query:    "nonexistent",
			expected: []models.Location{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locations, err := repo.SearchLocationsByText(ctx, tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, locations)
		})
	}
}