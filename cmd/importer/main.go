package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"geocoding-api/internal/config"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type LocationRecord struct {
	Prefecture   string
	Municipality string
	Address1     string
	Address2     string
	BlockLot     string
	Lat          float64
	Lon          float64
}

func main() {
	file := flag.String("file", "", "Path to the CSV file to import")
	flag.Parse()

	if *file == "" {
		fmt.Println("Error: --file flag is required")
		os.Exit(1)
	}

	fmt.Printf("Starting import from file: %s\n", *file)

	records, err := parseCSV(*file)
	if err != nil {
		fmt.Printf("Error parsing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d records\n", len(records))

	// Load config
	cfg, err := config.LoadConfig("configs")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Connect to DB
	conn, err := pgx.Connect(context.Background(), cfg.DBSource)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Ensure table exists
	err = createTableIfNotExists(conn)
	if err != nil {
		fmt.Printf("Error creating table: %v\n", err)
		os.Exit(1)
	}

	// Insert records
	err = insertRecords(conn, records)
	if err != nil {
		fmt.Printf("Error inserting records: %v\n", err)
		os.Exit(1)
	}

	// Verify data
	err = verifyImport(conn, len(records))
	if err != nil {
		fmt.Printf("Error verifying import: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully imported %d records\n", len(records))
}

func parseCSV(filePath string) ([]LocationRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var records []LocationRecord
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		if len(record) < 11 {
			return nil, fmt.Errorf("invalid record length: %d, expected at least 11 columns", len(record))
		}

		lat, err := strconv.ParseFloat(record[9], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude: %s", record[9])
		}

		lon, err := strconv.ParseFloat(record[10], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude: %s", record[10])
		}

		location := LocationRecord{
			Prefecture:   record[0],
			Municipality: record[1],
			Address1:     record[2],
			Address2:     record[3],
			BlockLot:     record[4],
			Lat:          lat,
			Lon:          lon,
		}

		records = append(records, location)
	}

	return records, nil
}

func createTableIfNotExists(conn *pgx.Conn) error {
	query := `
	CREATE TABLE IF NOT EXISTS locations (
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
	CREATE INDEX IF NOT EXISTS locations_geom_idx ON locations USING GIST (geom);
	CREATE INDEX IF NOT EXISTS locations_full_address_tsvector_idx ON locations USING GIN (full_address_tsvector);
	`
	_, err := conn.Exec(context.Background(), query)
	return err
}

func insertRecords(conn *pgx.Conn, records []LocationRecord) error {
	// Use CopyFrom for bulk insert
	_, err := conn.CopyFrom(
		context.Background(),
		pgx.Identifier{"locations"},
		[]string{"prefecture", "municipality", "address_1", "address_2", "block_lot", "geom"},
		pgx.CopyFromSlice(len(records), func(i int) ([]interface{}, error) {
			r := records[i]
			geom := fmt.Sprintf("SRID=4326;POINT(%f %f)", r.Lon, r.Lat) // PostGIS format: lon lat
			return []interface{}{r.Prefecture, r.Municipality, r.Address1, r.Address2, r.BlockLot, geom}, nil
		}),
	)
	return err
}

func verifyImport(conn *pgx.Conn, expectedCount int) error {
	var count int
	err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM locations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count records: %w", err)
	}

	if count != expectedCount {
		return fmt.Errorf("record count mismatch: expected %d, got %d", expectedCount, count)
	}

	// Check a sample geom
	var geom string
	err = conn.QueryRow(context.Background(), "SELECT ST_AsText(geom) FROM locations LIMIT 1").Scan(&geom)
	if err != nil {
		return fmt.Errorf("failed to check geom: %w", err)
	}

	fmt.Printf("Sample geom: %s\n", geom)
	return nil
}
