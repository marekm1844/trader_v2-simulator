package dataloader

import (
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/models"
)

// DataLoader defines the interface for loading and accessing historical price data
type DataLoader interface {
	// LoadMetadata loads the metadata from the data directory
	LoadMetadata() error

	// LoadCandles loads all candles for the given symbol and timeframe
	LoadCandles(symbol string, timeframe models.TimeFrame) error

	// GetCandles returns candles for the given symbol, timeframe and date range
	// If startDate or endDate is nil, the entire available range will be returned
	GetCandles(symbol string, timeframe models.TimeFrame, startDate, endDate *time.Time) ([]models.Candle, error)

	// GetCandleAt returns the candle for the given symbol, timeframe and timestamp
	GetCandleAt(symbol string, timeframe models.TimeFrame, timestamp time.Time) (*models.Candle, error)

	// GetSymbols returns all available symbols
	GetSymbols() []string

	// GetAvailableTimeframes returns all available timeframes for a symbol
	GetAvailableTimeframes(symbol string) []models.TimeFrame

	// GetDateRange returns the start and end date of available data for a symbol and timeframe
	GetDateRange(symbol string, timeframe models.TimeFrame) (startDate, endDate time.Time, err error)

	// Validate verifies the integrity and completeness of the loaded data
	Validate() error

	// IsDataAvailable checks if data is available for the given symbol, timeframe and date range
	IsDataAvailable(symbol string, timeframe models.TimeFrame, startDate, endDate *time.Time) bool
}