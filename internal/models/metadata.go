package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Metadata contains information about the available price data
type Metadata struct {
	Symbol            string    `json:"symbol"`             // Trading pair (e.g., "SOL-USD")
	PolygonSymbol     string    `json:"polygon_symbol"`     // Symbol format for Polygon API (e.g., "X:SOLUSD")
	StartDate         string    `json:"start_date"`         // First date with available data (YYYY-MM-DD)
	EndDate           string    `json:"end_date"`           // Last date with available data (YYYY-MM-DD)
	FetchTimestamp    string    `json:"fetch_timestamp"`    // When the data was last fetched
	DailyCandlesCount int       `json:"daily_candles_count"`  // Number of daily candles
	HourlyCandlesCount int      `json:"hourly_candles_count"` // Number of hourly candles
}

// Validate checks if the metadata is valid
func (m Metadata) Validate() error {
	if m.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}

	if m.PolygonSymbol == "" {
		return fmt.Errorf("polygon symbol cannot be empty")
	}

	// Check date formats
	layout := "2006-01-02"
	_, err := time.Parse(layout, m.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date format: %s", m.StartDate)
	}

	_, err = time.Parse(layout, m.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date format: %s", m.EndDate)
	}

	// Check counts
	if m.DailyCandlesCount < 0 {
		return fmt.Errorf("daily candles count cannot be negative: %d", m.DailyCandlesCount)
	}

	if m.HourlyCandlesCount < 0 {
		return fmt.Errorf("hourly candles count cannot be negative: %d", m.HourlyCandlesCount)
	}

	return nil
}

// LoadMetadata loads metadata from a file
func LoadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return &metadata, nil
}

// SaveMetadata saves metadata to a file
func SaveMetadata(metadata *Metadata, path string) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid metadata: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// UpdateCandleCounts updates the candle counts based on the available data
func (m *Metadata) UpdateCandleCounts(dailyCount, hourlyCount int) {
	m.DailyCandlesCount = dailyCount
	m.HourlyCandlesCount = hourlyCount
}

// SetTimeRange sets the start and end date based on the available data
func (m *Metadata) SetTimeRange(startDate, endDate string) {
	m.StartDate = startDate
	m.EndDate = endDate
}

// UpdateFetchTimestamp updates the fetch timestamp to the current time
func (m *Metadata) UpdateFetchTimestamp() {
	m.FetchTimestamp = time.Now().Format("2006-01-02 15:04:05")
}