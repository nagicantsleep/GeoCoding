-- Database setup script for GeoCoding API
-- This script creates the necessary tables, indexes, and extensions for full-text search and spatial queries

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- pgroonga can be added later if needed

-- Verify PostGIS installation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis') THEN
        RAISE EXCEPTION 'PostGIS extension is not available. Please install PostGIS.';
    END IF;
    RAISE NOTICE 'PostGIS extension verified successfully';
END
$$;

-- Create locations table with full-text search and spatial indexing
CREATE TABLE IF NOT EXISTS locations (
    id BIGSERIAL PRIMARY KEY,
    prefecture VARCHAR(255),
    municipality VARCHAR(255),
    address_1 VARCHAR(255),
    address_2 VARCHAR(255),
    block_lot VARCHAR(255),
    -- Full-text search vector
    full_address_tsvector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('simple', prefecture || ' ' || municipality || ' ' || address_1 || ' ' || address_2)
    ) STORED,
    -- PostGIS geography column for spatial queries (SRID 4326 = WGS84)
    geom GEOGRAPHY(POINT, 4326)
);

-- Create GIST index for spatial queries (nearest neighbor searches)
CREATE INDEX IF NOT EXISTS locations_geom_idx ON locations USING GIST (geom);

-- Create GIN index for full-text search
CREATE INDEX IF NOT EXISTS locations_full_address_tsvector_idx ON locations USING GIN (full_address_tsvector);

-- Create processed_files table for tracking imported CSV files
CREATE TABLE IF NOT EXISTS processed_files (
    id BIGSERIAL PRIMARY KEY,
    file_path TEXT UNIQUE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    record_count INTEGER NOT NULL
);

-- Create index on file_path for faster lookups
CREATE INDEX IF NOT EXISTS processed_files_path_idx ON processed_files (file_path);