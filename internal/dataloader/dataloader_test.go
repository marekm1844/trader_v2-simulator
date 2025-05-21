package dataloader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/models"
)

func TestFileDataLoader(t *testing.T) {
	// Create a temporary test directory
	testDir, err := os.MkdirTemp("", "dataloader_test")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create subdirectories
	dailyDir := filepath.Join(testDir, "daily")
	hourlyDir := filepath.Join(testDir, "hourly")
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		t.Fatalf("Failed to create daily directory: %v", err)
	}
	if err := os.MkdirAll(hourlyDir, 0755); err != nil {
		t.Fatalf("Failed to create hourly directory: %v", err)
	}

	// Create sample metadata
	metadata := &models.Metadata{
		Symbol:             "SOL-USD",
		PolygonSymbol:      "X:SOLUSD",
		StartDate:          "2025-04-25",
		EndDate:            "2025-05-18",
		FetchTimestamp:     "2025-05-16 11:00:00",
		DailyCandlesCount:  24,
		HourlyCandlesCount: 653,
	}

	// Save metadata
	metadataPath := filepath.Join(testDir, "metadata.json")
	if err := models.SaveMetadata(metadata, metadataPath); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Create sample daily candles
	dailyCandles := []models.Candle{
		{
			Timestamp: 1745539200, // 2025-04-25
			Open:      152.61,
			High:      157.08,
			Low:       150.01,
			Close:     150.91,
			Volume:    1724719.69,
		},
		{
			Timestamp: 1745625600, // 2025-04-26
			Open:      150.92,
			High:      155.45,
			Low:       148.77,
			Close:     152.33,
			Volume:    1521633.22,
		},
	}

	// Create sample hourly candles
	hourlyCandles := []models.Candle{
		{
			Timestamp: 1745539200, // 2025-04-25 00:00:00
			Open:      152.61,
			High:      153.42,
			Low:       151.98,
			Close:     152.11,
			Volume:    71862.48,
		},
		{
			Timestamp: 1745542800, // 2025-04-25 01:00:00
			Open:      152.13,
			High:      153.01,
			Low:       151.75,
			Close:     152.87,
			Volume:    68754.33,
		},
		{
			Timestamp: 1745546400, // 2025-04-25 02:00:00
			Open:      152.89,
			High:      154.25,
			Low:       152.44,
			Close:     153.76,
			Volume:    70123.21,
		},
	}

	// Save candles to files
	dailyCandlesPath := filepath.Join(dailyDir, "SOL-USD_daily.json")
	hourlyCandlesPath := filepath.Join(hourlyDir, "SOL-USD_hourly.json")

	dailyJSON, err := os.Create(dailyCandlesPath)
	if err != nil {
		t.Fatalf("Failed to create daily candles file: %v", err)
	}
	defer dailyJSON.Close()

	hourlyJSON, err := os.Create(hourlyCandlesPath)
	if err != nil {
		t.Fatalf("Failed to create hourly candles file: %v", err)
	}
	defer hourlyJSON.Close()

	// Write sample data
	if err := json.NewEncoder(dailyJSON).Encode(dailyCandles); err != nil {
		t.Fatalf("Failed to write daily candles: %v", err)
	}

	if err := json.NewEncoder(hourlyJSON).Encode(hourlyCandles); err != nil {
		t.Fatalf("Failed to write hourly candles: %v", err)
	}

	// Create the data loader
	loader := NewFileDataLoader(testDir)

	// Test LoadMetadata
	t.Run("LoadMetadata", func(t *testing.T) {
		err := loader.LoadMetadata()
		if err != nil {
			t.Fatalf("LoadMetadata() error = %v", err)
		}
	})

	// Test LoadCandles
	t.Run("LoadCandles", func(t *testing.T) {
		err := loader.LoadCandles("SOL-USD", models.TimeFrameDaily)
		if err != nil {
			t.Fatalf("LoadCandles(SOL-USD, daily) error = %v", err)
		}

		err = loader.LoadCandles("SOL-USD", models.TimeFrameHourly)
		if err != nil {
			t.Fatalf("LoadCandles(SOL-USD, hourly) error = %v", err)
		}
	})

	// Test GetCandles with no date range
	t.Run("GetCandles_NoDateRange", func(t *testing.T) {
		candles, err := loader.GetCandles("SOL-USD", models.TimeFrameDaily, nil, nil)
		if err != nil {
			t.Fatalf("GetCandles(SOL-USD, daily, nil, nil) error = %v", err)
		}
		if len(candles) != 2 {
			t.Errorf("Expected 2 daily candles, got %d", len(candles))
		}

		candles, err = loader.GetCandles("SOL-USD", models.TimeFrameHourly, nil, nil)
		if err != nil {
			t.Fatalf("GetCandles(SOL-USD, hourly, nil, nil) error = %v", err)
		}
		if len(candles) != 3 {
			t.Errorf("Expected 3 hourly candles, got %d", len(candles))
		}
	})

	// Test GetCandles with date range
	t.Run("GetCandles_WithDateRange", func(t *testing.T) {
		// Parse dates
		startDate := time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 4, 25, 23, 59, 59, 0, time.UTC)

		// Get candles for the first day only
		candles, err := loader.GetCandles("SOL-USD", models.TimeFrameDaily, &startDate, &endDate)
		if err != nil {
			t.Fatalf("GetCandles(SOL-USD, daily, 2025-04-25, 2025-04-25) error = %v", err)
		}
		if len(candles) != 1 {
			t.Errorf("Expected 1 daily candle, got %d", len(candles))
		}

		// Check first day hourly candles
		candles, err = loader.GetCandles("SOL-USD", models.TimeFrameHourly, &startDate, &endDate)
		if err != nil {
			t.Fatalf("GetCandles(SOL-USD, hourly, 2025-04-25, 2025-04-25) error = %v", err)
		}
		if len(candles) != 3 {
			t.Errorf("Expected 3 hourly candles, got %d", len(candles))
		}
	})

	// Test GetCandleAt
	t.Run("GetCandleAt", func(t *testing.T) {
		timestamp := time.Date(2025, 4, 25, 1, 0, 0, 0, time.UTC)
		candle, err := loader.GetCandleAt("SOL-USD", models.TimeFrameHourly, timestamp)
		if err != nil {
			t.Fatalf("GetCandleAt(SOL-USD, hourly, 2025-04-25 01:00:00) error = %v", err)
		}
		if candle == nil {
			t.Fatalf("Expected candle, got nil")
		}
		if candle.Close != 152.87 {
			t.Errorf("Expected close price 152.87, got %.2f", candle.Close)
		}
	})

	// Test GetSymbols
	t.Run("GetSymbols", func(t *testing.T) {
		symbols := loader.GetSymbols()
		if len(symbols) != 1 {
			t.Errorf("Expected 1 symbol, got %d", len(symbols))
		}
		if symbols[0] != "SOL-USD" {
			t.Errorf("Expected symbol SOL-USD, got %s", symbols[0])
		}
	})

	// Test GetAvailableTimeframes
	t.Run("GetAvailableTimeframes", func(t *testing.T) {
		timeframes := loader.GetAvailableTimeframes("SOL-USD")
		if len(timeframes) != 2 {
			t.Errorf("Expected 2 timeframes, got %d", len(timeframes))
		}
	})

	// Test GetDateRange
	t.Run("GetDateRange", func(t *testing.T) {
		start, end, err := loader.GetDateRange("SOL-USD", models.TimeFrameDaily)
		if err != nil {
			t.Fatalf("GetDateRange(SOL-USD, daily) error = %v", err)
		}
		
		expectedStart := time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
		
		if !start.Equal(expectedStart) {
			t.Errorf("Expected start date %s, got %s", expectedStart, start)
		}
		
		if !end.Equal(expectedEnd) {
			t.Errorf("Expected end date %s, got %s", expectedEnd, end)
		}
	})

	// Test IsDataAvailable
	t.Run("IsDataAvailable", func(t *testing.T) {
		// Data should be available for the full range
		startDate := time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
		
		if !loader.IsDataAvailable("SOL-USD", models.TimeFrameDaily, &startDate, &endDate) {
			t.Errorf("Expected data to be available for SOL-USD, daily from 2025-04-25 to 2025-05-18")
		}
		
		// Data should not be available for a range outside the available data
		invalidStartDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		invalidEndDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		
		if loader.IsDataAvailable("SOL-USD", models.TimeFrameDaily, &invalidStartDate, &invalidEndDate) {
			t.Errorf("Expected data to NOT be available for SOL-USD, daily from 2025-01-01 to 2025-01-31")
		}
	})
}