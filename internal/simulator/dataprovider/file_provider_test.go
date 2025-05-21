package dataprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
)

func TestFileDataProvider(t *testing.T) {
	// Use the actual data directory for integration testing
	dataDir := "../../../data"
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Import NewFileDataProvider implementation directly
	metadataPath := filepath.Join(absDataDir, "metadata.json")
	
	// Check if metadata file exists
	if _, statErr := os.Stat(metadataPath); os.IsNotExist(statErr) {
		t.Fatalf("metadata.json not found in %s", absDataDir)
	}
	
	// Create the provider the same way the constructor does
	provider := &FileDataProvider{
		dataDir:       absDataDir,
		metadataPath:  metadataPath,
		dailyCandles:  make(map[string][]simulator.Candle),
		hourlyCandles: make(map[string][]simulator.Candle),
		dataRanges:    make(map[string]map[simulator.TimeFrame]struct{ start, end time.Time }),
	}
	
	// Initialize by reloading data
	err = provider.Reload(context.Background())
	if err != nil {
		t.Fatalf("Failed to create FileDataProvider: %v", err)
	}

	// Test GetAvailableSymbols
	t.Run("GetAvailableSymbols", func(t *testing.T) {
		symbols, err := provider.GetAvailableSymbols(context.Background())
		if err != nil {
			t.Fatalf("GetAvailableSymbols failed: %v", err)
		}

		if len(symbols) == 0 {
			t.Error("Expected at least one symbol, got none")
		}

		// Check that SOL-USD is in the list
		hasSOL := false
		for _, s := range symbols {
			if s == "SOL-USD" {
				hasSOL = true
				break
			}
		}
		if !hasSOL {
			t.Error("Expected SOL-USD in available symbols")
		}
	})

	// Test GetDataRange
	t.Run("GetDataRange", func(t *testing.T) {
		// Test daily timeframe
		startDaily, endDaily, err := provider.GetDataRange(context.Background(), "SOL-USD", simulator.TimeFrameDaily)
		if err != nil {
			t.Fatalf("GetDataRange for daily failed: %v", err)
		}

		if startDaily.IsZero() || endDaily.IsZero() {
			t.Error("Expected non-zero start and end times for daily timeframe")
		}

		// Test hourly timeframe
		startHourly, endHourly, err := provider.GetDataRange(context.Background(), "SOL-USD", simulator.TimeFrameHourly)
		if err != nil {
			t.Fatalf("GetDataRange for hourly failed: %v", err)
		}

		if startHourly.IsZero() || endHourly.IsZero() {
			t.Error("Expected non-zero start and end times for hourly timeframe")
		}

		// Check that hourly data range contains daily range
		if startHourly.After(startDaily) || endHourly.Before(endDaily) {
			t.Errorf("Expected hourly range to contain daily range, got hourly=%v-%v, daily=%v-%v",
				startHourly, endHourly, startDaily, endDaily)
		}
	})

	// Test GetCandles
	t.Run("GetCandles", func(t *testing.T) {
		// Get data range first
		startDaily, dateEnd, err := provider.GetDataRange(context.Background(), "SOL-USD", simulator.TimeFrameDaily)
		if err != nil {
			t.Fatalf("GetDataRange for daily failed: %v", err)
		}

		// Calculate request end time (using the minimum of start+24h or the actual data end)
		requestEnd := startDaily.Add(24 * time.Hour)
		if requestEnd.After(dateEnd) {
			requestEnd = dateEnd
		}
		candles, err := provider.GetCandles(context.Background(), "SOL-USD", simulator.TimeFrameDaily, startDaily, requestEnd)
		if err != nil {
			t.Fatalf("GetCandles failed: %v", err)
		}

		if len(candles) == 0 {
			t.Error("Expected at least one candle, got none")
		}

		// Check candle properties
		for i, candle := range candles {
			if candle.Symbol != "SOL-USD" {
				t.Errorf("Candle %d has wrong symbol: expected SOL-USD, got %s", i, candle.Symbol)
			}

			if candle.Timestamp.Before(startDaily) || candle.Timestamp.Equal(requestEnd) || candle.Timestamp.After(requestEnd) {
				t.Errorf("Candle %d has timestamp outside requested range: %v", i, candle.Timestamp)
			}

			if candle.Open <= 0 || candle.High <= 0 || candle.Low <= 0 || candle.Close <= 0 {
				t.Errorf("Candle %d has invalid price data: open=%.2f, high=%.2f, low=%.2f, close=%.2f",
					i, candle.Open, candle.High, candle.Low, candle.Close)
			}
		}

		// Get hourly candles
		startHourly, _, err := provider.GetDataRange(context.Background(), "SOL-USD", simulator.TimeFrameHourly)
		if err != nil {
			t.Fatalf("GetDataRange for hourly failed: %v", err)
		}

		requestEnd = startHourly.Add(5 * time.Hour)
		hourlyCandles, err := provider.GetCandles(context.Background(), "SOL-USD", simulator.TimeFrameHourly, startHourly, requestEnd)
		if err != nil {
			t.Fatalf("GetCandles for hourly failed: %v", err)
		}

		if len(hourlyCandles) == 0 {
			t.Error("Expected at least one hourly candle, got none")
		}

		if len(hourlyCandles) > 5 {
			t.Errorf("Expected at most 5 hourly candles, got %d", len(hourlyCandles))
		}
	})

	// Test invalid requests
	t.Run("InvalidRequests", func(t *testing.T) {
		// Test invalid symbol
		_, _, err := provider.GetDataRange(context.Background(), "INVALID-SYMBOL", simulator.TimeFrameDaily)
		if err == nil {
			t.Error("Expected error for invalid symbol, got nil")
		}

		// Test invalid timeframe
		_, _, err = provider.GetDataRange(context.Background(), "SOL-USD", "invalid")
		if err == nil {
			t.Error("Expected error for invalid timeframe, got nil")
		}
	})
}

