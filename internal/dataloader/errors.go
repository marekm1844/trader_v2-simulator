package dataloader

import (
	"fmt"
)

// ErrDataNotLoaded is returned when attempting to access data that hasn't been loaded
type ErrDataNotLoaded struct {
	Symbol    string
	Timeframe string
}

func (e ErrDataNotLoaded) Error() string {
	return fmt.Sprintf("data not loaded for symbol=%s, timeframe=%s", e.Symbol, e.Timeframe)
}

// ErrSymbolNotFound is returned when the requested symbol is not found
type ErrSymbolNotFound struct {
	Symbol string
}

func (e ErrSymbolNotFound) Error() string {
	return fmt.Sprintf("symbol not found: %s", e.Symbol)
}

// ErrTimeframeNotSupported is returned when the requested timeframe is not supported
type ErrTimeframeNotSupported struct {
	Symbol    string
	Timeframe string
}

func (e ErrTimeframeNotSupported) Error() string {
	return fmt.Sprintf("timeframe %s not supported for symbol %s", e.Timeframe, e.Symbol)
}

// ErrNoDataInRange is returned when no data is available in the specified date range
type ErrNoDataInRange struct {
	Symbol    string
	Timeframe string
	StartDate string
	EndDate   string
}

func (e ErrNoDataInRange) Error() string {
	return fmt.Sprintf("no data available for symbol=%s, timeframe=%s in range %s to %s", 
		e.Symbol, e.Timeframe, e.StartDate, e.EndDate)
}

// ErrInvalidDataFormat is returned when the data file has an invalid format
type ErrInvalidDataFormat struct {
	FilePath string
	Cause    error
}

func (e ErrInvalidDataFormat) Error() string {
	return fmt.Sprintf("invalid data format in file %s: %v", e.FilePath, e.Cause)
}

// ErrDataInconsistency is returned when data validation fails
type ErrDataInconsistency struct {
	Message string
}

func (e ErrDataInconsistency) Error() string {
	return fmt.Sprintf("data inconsistency detected: %s", e.Message)
}