package models

import (
	"fmt"
	"time"
)

// Candle represents a single price candlestick
type Candle struct {
	Timestamp int64   `json:"timestamp"` // Unix timestamp in seconds
	Open      float64 `json:"open"`      // Opening price
	High      float64 `json:"high"`      // Highest price during the period
	Low       float64 `json:"low"`       // Lowest price during the period
	Close     float64 `json:"close"`     // Closing price
	Volume    float64 `json:"volume"`    // Trading volume
}

// Time returns the timestamp as a time.Time
func (c Candle) Time() time.Time {
	return time.Unix(c.Timestamp, 0)
}

// FormattedTime returns the timestamp as a formatted string
func (c Candle) FormattedTime(layout string) string {
	return c.Time().Format(layout)
}

// DateString returns the date in YYYY-MM-DD format
func (c Candle) DateString() string {
	return c.Time().Format("2006-01-02")
}

// String returns a readable representation of the candle
func (c Candle) String() string {
	return fmt.Sprintf(
		"Candle{time: %s, open: %.2f, high: %.2f, low: %.2f, close: %.2f, volume: %.2f}",
		c.Time().Format(time.RFC3339),
		c.Open, c.High, c.Low, c.Close, c.Volume,
	)
}

// Validate ensures the candle data is consistent
func (c Candle) Validate() error {
	if c.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", c.Timestamp)
	}

	if c.Low > c.High {
		return fmt.Errorf("low price (%.2f) cannot be greater than high price (%.2f)", c.Low, c.High)
	}

	if c.Open < c.Low || c.Open > c.High {
		return fmt.Errorf("open price (%.2f) must be between low (%.2f) and high (%.2f)", c.Open, c.Low, c.High)
	}

	if c.Close < c.Low || c.Close > c.High {
		return fmt.Errorf("close price (%.2f) must be between low (%.2f) and high (%.2f)", c.Close, c.Low, c.High)
	}

	if c.Volume < 0 {
		return fmt.Errorf("volume cannot be negative: %.2f", c.Volume)
	}

	return nil
}

// CandleList represents a list of candles for a symbol and timeframe
type CandleList struct {
	Symbol    string   `json:"symbol"`
	TimeFrame string   `json:"timeframe"`
	Candles   []Candle `json:"candles"`
}

// Sort sorts the candles by timestamp in ascending order
func (cl *CandleList) Sort() {
	// Implementation would use the sort package
	// This is a placeholder for the actual implementation
}