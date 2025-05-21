package dataloader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/models"
)

// FileDataLoader implements the DataLoader interface using file-based storage
type FileDataLoader struct {
	dataDir     string
	metadata    *models.Metadata
	metadataMap map[string]*models.Metadata
	candleCache map[string]map[models.TimeFrame][]models.Candle
	mu          sync.RWMutex
}

// NewFileDataLoader creates a new FileDataLoader with the specified data directory
func NewFileDataLoader(dataDir string) *FileDataLoader {
	return &FileDataLoader{
		dataDir:     dataDir,
		metadataMap: make(map[string]*models.Metadata),
		candleCache: make(map[string]map[models.TimeFrame][]models.Candle),
	}
}

// LoadMetadata loads the metadata from the data directory
func (l *FileDataLoader) LoadMetadata() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	metadataPath := filepath.Join(l.dataDir, "metadata.json")
	
	// Reset metadata
	l.metadata = nil
	l.metadataMap = make(map[string]*models.Metadata)

	// Load global metadata
	metadata, err := models.LoadMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load global metadata: %w", err)
	}
	
	l.metadata = metadata
	l.metadataMap[metadata.Symbol] = metadata

	return nil
}

// getFilePath returns the file path for the given symbol and timeframe
func (l *FileDataLoader) getFilePath(symbol string, timeframe models.TimeFrame) (string, error) {
	if !timeframe.IsValid() {
		return "", fmt.Errorf("invalid timeframe: %s", timeframe)
	}

	filename := fmt.Sprintf("%s_%s.json", symbol, timeframe)
	return filepath.Join(l.dataDir, timeframe.Directory(), filename), nil
}

// LoadCandles loads all candles for the given symbol and timeframe
func (l *FileDataLoader) LoadCandles(symbol string, timeframe models.TimeFrame) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if metadata is loaded
	if l.metadata == nil {
		return fmt.Errorf("metadata not loaded, call LoadMetadata first")
	}

	// Check if symbol exists in metadata
	if _, ok := l.metadataMap[symbol]; !ok {
		return fmt.Errorf("symbol not found in metadata: %s", symbol)
	}

	// Get file path
	filePath, err := l.getFilePath(symbol, timeframe)
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("candle file not found: %s", filePath)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read candle file: %w", err)
	}

	// Parse the candles
	var candles []models.Candle
	if err := json.Unmarshal(data, &candles); err != nil {
		return fmt.Errorf("failed to unmarshal candles: %w", err)
	}

	// Validate candles
	for i, candle := range candles {
		if err := candle.Validate(); err != nil {
			return fmt.Errorf("invalid candle at index %d: %w", i, err)
		}
	}

	// Initialize the cache for this symbol if needed
	if _, ok := l.candleCache[symbol]; !ok {
		l.candleCache[symbol] = make(map[models.TimeFrame][]models.Candle)
	}

	// Cache the candles
	l.candleCache[symbol][timeframe] = candles

	return nil
}

// GetCandles returns candles for the given symbol, timeframe and date range
func (l *FileDataLoader) GetCandles(symbol string, timeframe models.TimeFrame, startDate, endDate *time.Time) ([]models.Candle, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if the candles are already loaded
	if _, ok := l.candleCache[symbol]; !ok || len(l.candleCache[symbol][timeframe]) == 0 {
		l.mu.RUnlock()
		if err := l.LoadCandles(symbol, timeframe); err != nil {
			return nil, err
		}
		l.mu.RLock()
	}

	// Get all candles for the symbol and timeframe
	allCandles := l.candleCache[symbol][timeframe]
	if len(allCandles) == 0 {
		return nil, fmt.Errorf("no candles found for symbol %s and timeframe %s", symbol, timeframe)
	}

	// If no date range is specified, return all candles
	if startDate == nil && endDate == nil {
		return allCandles, nil
	}

	// Filter candles by date range
	var filteredCandles []models.Candle
	for _, candle := range allCandles {
		candleTime := candle.Time()
		
		// Skip candles before start date
		if startDate != nil && candleTime.Before(*startDate) {
			continue
		}
		
		// Skip candles after end date
		if endDate != nil && candleTime.After(*endDate) {
			continue
		}
		
		filteredCandles = append(filteredCandles, candle)
	}

	if len(filteredCandles) == 0 {
		return nil, fmt.Errorf("no candles found for symbol %s and timeframe %s in the specified date range", symbol, timeframe)
	}

	return filteredCandles, nil
}

// GetCandleAt returns the candle for the given symbol, timeframe and timestamp
func (l *FileDataLoader) GetCandleAt(symbol string, timeframe models.TimeFrame, timestamp time.Time) (*models.Candle, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if the candles are already loaded
	if _, ok := l.candleCache[symbol]; !ok || len(l.candleCache[symbol][timeframe]) == 0 {
		l.mu.RUnlock()
		if err := l.LoadCandles(symbol, timeframe); err != nil {
			return nil, err
		}
		l.mu.RLock()
	}

	// Find the candle with the closest timestamp
	targetTimestamp := timestamp.Unix()
	
	// For daily timeframe, we only care about the date
	if timeframe == models.TimeFrameDaily {
		targetTimestamp = time.Date(
			timestamp.Year(), timestamp.Month(), timestamp.Day(),
			0, 0, 0, 0, timestamp.Location(),
		).Unix()
	}

	for _, candle := range l.candleCache[symbol][timeframe] {
		if candle.Timestamp == targetTimestamp {
			return &candle, nil
		}
	}

	return nil, fmt.Errorf("no candle found for symbol %s, timeframe %s at timestamp %s", 
		symbol, timeframe, timestamp.Format(time.RFC3339))
}

// GetSymbols returns all available symbols
func (l *FileDataLoader) GetSymbols() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var symbols []string
	for symbol := range l.metadataMap {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetAvailableTimeframes returns all available timeframes for a symbol
func (l *FileDataLoader) GetAvailableTimeframes(symbol string) []models.TimeFrame {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var timeframes []models.TimeFrame
	
	metadata, ok := l.metadataMap[symbol]
	if !ok {
		return timeframes
	}
	
	if metadata.DailyCandlesCount > 0 {
		timeframes = append(timeframes, models.TimeFrameDaily)
	}
	
	if metadata.HourlyCandlesCount > 0 {
		timeframes = append(timeframes, models.TimeFrameHourly)
	}
	
	return timeframes
}

// GetDateRange returns the start and end date of available data for a symbol and timeframe
func (l *FileDataLoader) GetDateRange(symbol string, timeframe models.TimeFrame) (startDate, endDate time.Time, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	metadata, ok := l.metadataMap[symbol]
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("symbol not found in metadata: %s", symbol)
	}

	// Parse dates from metadata
	layout := "2006-01-02"
	
	start, err := time.Parse(layout, metadata.StartDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start date in metadata: %w", err)
	}
	
	end, err := time.Parse(layout, metadata.EndDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end date in metadata: %w", err)
	}
	
	return start, end, nil
}

// Validate verifies the integrity and completeness of the loaded data
func (l *FileDataLoader) Validate() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.metadata == nil {
		return fmt.Errorf("metadata not loaded")
	}

	for symbol, metadata := range l.metadataMap {
		// Check if we have the expected number of candles
		if metadata.DailyCandlesCount > 0 {
			if _, ok := l.candleCache[symbol][models.TimeFrameDaily]; !ok {
				// Load the candles if they're not in the cache
				l.mu.RUnlock()
				if err := l.LoadCandles(symbol, models.TimeFrameDaily); err != nil {
					return fmt.Errorf("failed to load daily candles for %s: %w", symbol, err)
				}
				l.mu.RLock()
			}
			
			if len(l.candleCache[symbol][models.TimeFrameDaily]) != metadata.DailyCandlesCount {
				return fmt.Errorf(
					"daily candle count mismatch for %s: expected %d, got %d",
					symbol,
					metadata.DailyCandlesCount,
					len(l.candleCache[symbol][models.TimeFrameDaily]),
				)
			}
		}
		
		if metadata.HourlyCandlesCount > 0 {
			if _, ok := l.candleCache[symbol][models.TimeFrameHourly]; !ok {
				// Load the candles if they're not in the cache
				l.mu.RUnlock()
				if err := l.LoadCandles(symbol, models.TimeFrameHourly); err != nil {
					return fmt.Errorf("failed to load hourly candles for %s: %w", symbol, err)
				}
				l.mu.RLock()
			}
			
			if len(l.candleCache[symbol][models.TimeFrameHourly]) != metadata.HourlyCandlesCount {
				return fmt.Errorf(
					"hourly candle count mismatch for %s: expected %d, got %d",
					symbol,
					metadata.HourlyCandlesCount,
					len(l.candleCache[symbol][models.TimeFrameHourly]),
				)
			}
		}
	}

	return nil
}

// IsDataAvailable checks if data is available for the given symbol, timeframe and date range
func (l *FileDataLoader) IsDataAvailable(symbol string, timeframe models.TimeFrame, startDate, endDate *time.Time) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if metadata is loaded
	if l.metadata == nil {
		return false
	}

	// Check if symbol exists in metadata
	metadata, ok := l.metadataMap[symbol]
	if !ok {
		return false
	}

	// Check if timeframe has data
	if timeframe == models.TimeFrameDaily && metadata.DailyCandlesCount == 0 {
		return false
	}
	if timeframe == models.TimeFrameHourly && metadata.HourlyCandlesCount == 0 {
		return false
	}

	// If no date range is specified, we only need to check if the candles exist
	if startDate == nil && endDate == nil {
		return true
	}

	// Parse metadata dates
	layout := "2006-01-02"
	
	metadataStart, err := time.Parse(layout, metadata.StartDate)
	if err != nil {
		return false
	}
	
	metadataEnd, err := time.Parse(layout, metadata.EndDate)
	if err != nil {
		return false
	}

	// Check if requested date range is within available data
	if startDate != nil && startDate.After(metadataEnd) {
		return false
	}
	
	if endDate != nil && endDate.Before(metadataStart) {
		return false
	}

	return true
}