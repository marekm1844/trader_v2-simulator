package dataprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/models"
	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
)

// FileDataProvider implements the simulator.DataProvider interface
// using files from the local filesystem
type FileDataProvider struct {
	dataDir      string                                     // Root directory for data files
	metadataPath string                                     // Path to metadata.json
	mu           sync.RWMutex                               // Mutex for thread safety
	metadata     *models.Metadata                           // Cached metadata
	dailyCandles map[string][]simulator.Candle              // Cached daily candles by symbol
	hourlyCandles map[string][]simulator.Candle             // Cached hourly candles by symbol
	dataRanges   map[string]map[simulator.TimeFrame]struct{ start, end time.Time } // Cached data ranges
}

// FileCandle represents the structure of candle data stored in JSON files
type FileCandle struct {
	Sequence  int       `json:"sequence"`
	Timestamp int64     `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Date      string    `json:"date,omitempty"`      // For daily candles
	DateTime  string    `json:"datetime,omitempty"`  // For hourly candles
}

// NewFileDataProvider creates a new FileDataProvider instance
func NewFileDataProvider(dataDir string) (*FileDataProvider, error) {
	metadataPath := filepath.Join(dataDir, "metadata.json")
	
	provider := &FileDataProvider{
		dataDir:      dataDir,
		metadataPath: metadataPath,
		dailyCandles: make(map[string][]simulator.Candle),
		hourlyCandles: make(map[string][]simulator.Candle),
		dataRanges:   make(map[string]map[simulator.TimeFrame]struct{ start, end time.Time }),
	}
	
	// Load initial data
	if err := provider.Reload(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load initial data: %w", err)
	}
	
	return provider, nil
}

// GetCandles fetches candles for a specific symbol, timeframe, and date range
func (p *FileDataProvider) GetCandles(ctx context.Context, symbol string, timeframe simulator.TimeFrame, 
	startTime, endTime time.Time) ([]simulator.Candle, error) {
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var candles []simulator.Candle
	
	switch timeframe {
	case simulator.TimeFrameDaily:
		if c, ok := p.dailyCandles[symbol]; ok {
			candles = c
		} else {
			return nil, fmt.Errorf("no daily candles found for symbol %s", symbol)
		}
	case simulator.TimeFrameHourly:
		if c, ok := p.hourlyCandles[symbol]; ok {
			candles = c
		} else {
			return nil, fmt.Errorf("no hourly candles found for symbol %s", symbol)
		}
	default:
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}
	
	// Filter candles by time range
	filtered := make([]simulator.Candle, 0)
	for _, candle := range candles {
		if (candle.Timestamp.Equal(startTime) || candle.Timestamp.After(startTime)) &&
			candle.Timestamp.Before(endTime) {
			filtered = append(filtered, candle)
		}
	}
	
	return filtered, nil
}

// GetAvailableSymbols returns all available trading symbols
func (p *FileDataProvider) GetAvailableSymbols(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.metadata == nil {
		return nil, fmt.Errorf("metadata not loaded")
	}
	
	// Currently we only support the single symbol in metadata
	return []string{p.metadata.Symbol}, nil
}

// GetDataRange returns the available data range for a symbol and timeframe
func (p *FileDataProvider) GetDataRange(ctx context.Context, symbol string, timeframe simulator.TimeFrame) (startTime, endTime time.Time, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if ranges, ok := p.dataRanges[symbol]; ok {
		if timeRange, ok := ranges[timeframe]; ok {
			return timeRange.start, timeRange.end, nil
		}
	}
	
	return time.Time{}, time.Time{}, fmt.Errorf("no data range found for symbol %s and timeframe %s", symbol, timeframe)
}

// Reload refreshes data from the underlying source
func (p *FileDataProvider) Reload(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Load metadata
	metadata, err := models.LoadMetadata(p.metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}
	p.metadata = metadata
	
	symbol := metadata.Symbol
	
	// Initialize data ranges map for this symbol
	if _, ok := p.dataRanges[symbol]; !ok {
		p.dataRanges[symbol] = make(map[simulator.TimeFrame]struct{ start, end time.Time })
	}
	
	// Load daily candles
	dailyPath := filepath.Join(p.dataDir, "daily", fmt.Sprintf("%s_daily.json", symbol))
	dailyCandles, startDaily, endDaily, err := p.loadCandlesFromFile(dailyPath, symbol)
	if err != nil {
		return fmt.Errorf("failed to load daily candles: %w", err)
	}
	p.dailyCandles[symbol] = dailyCandles
	p.dataRanges[symbol][simulator.TimeFrameDaily] = struct{ start, end time.Time }{startDaily, endDaily}
	
	// Load hourly candles
	hourlyPath := filepath.Join(p.dataDir, "hourly", fmt.Sprintf("%s_hourly.json", symbol))
	hourlyCandles, startHourly, endHourly, err := p.loadCandlesFromFile(hourlyPath, symbol)
	if err != nil {
		return fmt.Errorf("failed to load hourly candles: %w", err)
	}
	p.hourlyCandles[symbol] = hourlyCandles
	p.dataRanges[symbol][simulator.TimeFrameHourly] = struct{ start, end time.Time }{startHourly, endHourly}
	
	return nil
}

// loadCandlesFromFile loads candles from a JSON file
func (p *FileDataProvider) loadCandlesFromFile(filePath, symbol string) ([]simulator.Candle, time.Time, time.Time, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, time.Time{}, time.Time{}, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	var fileCandles []FileCandle
	if err := json.Unmarshal(data, &fileCandles); err != nil {
		return nil, time.Time{}, time.Time{}, fmt.Errorf("failed to unmarshal candles from %s: %w", filePath, err)
	}
	
	if len(fileCandles) == 0 {
		return nil, time.Time{}, time.Time{}, fmt.Errorf("no candles found in %s", filePath)
	}
	
	// Convert to simulator.Candle format
	candles := make([]simulator.Candle, len(fileCandles))
	for i, fc := range fileCandles {
		// Convert millisecond timestamp to time.Time
		timestamp := time.UnixMilli(fc.Timestamp)
		
		candles[i] = simulator.Candle{
			Timestamp: timestamp,
			Symbol:    symbol,
			Open:      fc.Open,
			High:      fc.High,
			Low:       fc.Low,
			Close:     fc.Close,
			Volume:    fc.Volume,
		}
	}
	
	// Determine start and end times
	startTime := candles[0].Timestamp
	endTime := candles[len(candles)-1].Timestamp
	
	return candles, startTime, endTime, nil
}