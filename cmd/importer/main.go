package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"geocoding-api/internal/config"
	"os"
	"path/filepath"
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
	directory := flag.String("directory", "", "Path to the directory containing CSV files to import")
	flag.Parse()

	if *file == "" && *directory == "" {
		fmt.Println("Error: either --file or --directory flag is required")
		os.Exit(1)
	}

	if *file != "" && *directory != "" {
		fmt.Println("Error: cannot specify both --file and --directory flags")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.LoadConfig(filepath.Join(".", "configs"))
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

	// Ensure tables exist
	err = createTablesIfNotExists(conn)
	if err != nil {
		fmt.Printf("Error creating tables: %v\n", err)
		os.Exit(1)
	}

	var totalRecords int
	var processedFiles int
	var failedFiles int

	if *file != "" {
		// Single file import (backward compatibility)
		fmt.Printf("Starting import from file: %s\n", *file)

		records, err := parseCSV(*file)
		if err != nil {
			fmt.Printf("Error parsing CSV: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Parsed %d records\n", len(records))

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
	} else {
		// Directory import
		fmt.Printf("Starting import from directory: %s\n", *directory)

		files, err := findCSVFiles(*directory)
		if err != nil {
			fmt.Printf("Error finding CSV files: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d CSV files to process\n", len(files))

		for _, filePath := range files {
			fmt.Printf("Processing file: %s\n", filePath)

			// Check if file has been processed
			processed, err := isFileProcessed(conn, filePath)
			if err != nil {
				fmt.Printf("Error checking if file processed: %v\n", err)
				failedFiles++
				continue
			}

			if processed {
				fmt.Printf("Skipping already processed file: %s\n", filePath)
				continue
			}

			records, err := parseCSV(filePath)
			if err != nil {
				fmt.Printf("Error parsing CSV %s: %v\n", filePath, err)
				failedFiles++
				continue
			}

			fmt.Printf("Parsed %d records from %s\n", len(records), filePath)

			// Insert records
			err = insertRecords(conn, records)
			if err != nil {
				fmt.Printf("Error inserting records from %s: %v\n", filePath, err)
				failedFiles++
				continue
			}

			// Mark file as processed
			err = markFileProcessed(conn, filePath, len(records))
			if err != nil {
				fmt.Printf("Error marking file as processed: %v\n", err)
				// Don't increment failedFiles here as the data was inserted successfully
			}

			totalRecords += len(records)
			processedFiles++
			fmt.Printf("Successfully processed %s (%d records)\n", filePath, len(records))
		}

		fmt.Printf("Directory import completed: %d files processed, %d files failed, %d total records imported\n", processedFiles, failedFiles, totalRecords)
	}
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

func createTablesIfNotExists(conn *pgx.Conn) error {
	// Create locations table
	locationsQuery := `
	CREATE TABLE IF NOT EXISTS locations (
		id BIGSERIAL PRIMARY KEY,
		prefecture VARCHAR(255),
		municipality VARCHAR(255),
		address_1 VARCHAR(255),
		address_2 VARCHAR(255),
		block_lot VARCHAR(255),
		full_address_tsvector TSVECTOR GENERATED ALWAYS AS (
			to_tsvector('japanese', prefecture || ' ' || municipality || ' ' || address_1 || ' ' || address_2)
		) STORED,
		geom GEOGRAPHY(POINT, 4326)
	);
	CREATE INDEX IF NOT EXISTS locations_geom_idx ON locations USING GIST (geom);
	CREATE INDEX IF NOT EXISTS locations_full_address_tsvector_idx ON locations USING GIN (full_address_tsvector);
	`
	_, err := conn.Exec(context.Background(), locationsQuery)
	if err != nil {
		return err
	}

	// Create processed_files table
	processedFilesQuery := `
	CREATE TABLE IF NOT EXISTS processed_files (
		id BIGSERIAL PRIMARY KEY,
		file_path TEXT UNIQUE NOT NULL,
		processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		record_count INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS processed_files_path_idx ON processed_files (file_path);
	`
	_, err = conn.Exec(context.Background(), processedFilesQuery)
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

	if count < expectedCount {
		return fmt.Errorf("record count mismatch: expected at least %d, got %d", expectedCount, count)
	}

	// Check a sample geom
	var geom string
	err = conn.QueryRow(context.Background(), "SELECT ST_AsText(geom) FROM locations LIMIT 1").Scan(&geom)
	if err != nil {
		return fmt.Errorf("failed to check geom: %w", err)
	}

	fmt.Printf("Sample geom: %s\n", geom)
	fmt.Printf("✓ Verified: %d total records in database (imported %d new records)\n", count, expectedCount)
	return nil
}

func findCSVFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".csv" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isFileProcessed(conn *pgx.Conn, filePath string) (bool, error) {
	var exists bool
	err := conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM processed_files WHERE file_path = $1)",
		filePath).Scan(&exists)
	return exists, err
}

func markFileProcessed(conn *pgx.Conn, filePath string, recordCount int) error {
	_, err := conn.Exec(context.Background(),
		"INSERT INTO processed_files (file_path, record_count) VALUES ($1, $2) ON CONFLICT (file_path) DO NOTHING",
		filePath, recordCount)
	return err
}
