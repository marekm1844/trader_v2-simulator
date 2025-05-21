package models

import (
	"testing"
	"time"
)

func TestSymbol(t *testing.T) {
	t.Run("NewSymbol creates valid symbol", func(t *testing.T) {
		sym, err := NewSymbol("SOL-USD")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if sym.Base != "SOL" {
			t.Errorf("Expected Base to be SOL, got %s", sym.Base)
		}
		if sym.Quote != "USD" {
			t.Errorf("Expected Quote to be USD, got %s", sym.Quote)
		}
	})

	t.Run("NewSymbol validates input", func(t *testing.T) {
		invalid := []string{
			"SOL", "SOL-", "-USD", "SOL_USD", "",
		}
		for _, s := range invalid {
			sym, err := NewSymbol(s)
			if err == nil {
				t.Errorf("Expected error for invalid symbol %s, got none. Symbol: %+v", s, sym)
			}
		}
	})

	t.Run("Symbol.String returns correct format", func(t *testing.T) {
		sym := Symbol{Base: "SOL", Quote: "USD"}
		if sym.String() != "SOL-USD" {
			t.Errorf("Expected SOL-USD, got %s", sym.String())
		}
	})

	t.Run("Symbol.PolygonFormat returns correct format", func(t *testing.T) {
		sym := Symbol{Base: "SOL", Quote: "USD"}
		if sym.PolygonFormat() != "X:SOLUSD" {
			t.Errorf("Expected X:SOLUSD, got %s", sym.PolygonFormat())
		}
	})
}

func TestTimeFrame(t *testing.T) {
	t.Run("TimeFrame.IsValid validates time frames", func(t *testing.T) {
		valid := []TimeFrame{TimeFrameDaily, TimeFrameHourly}
		for _, tf := range valid {
			if !tf.IsValid() {
				t.Errorf("Expected %s to be valid, but it's not", tf)
			}
		}

		invalid := []TimeFrame{"", "minute", "weekly"}
		for _, tf := range invalid {
			if tf.IsValid() {
				t.Errorf("Expected %s to be invalid, but it's valid", tf)
			}
		}
	})

	t.Run("ParseTimeFrame parses valid time frames", func(t *testing.T) {
		tf, err := ParseTimeFrame("daily")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if tf != TimeFrameDaily {
			t.Errorf("Expected TimeFrameDaily, got %s", tf)
		}

		tf, err = ParseTimeFrame("hourly")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if tf != TimeFrameHourly {
			t.Errorf("Expected TimeFrameHourly, got %s", tf)
		}
	})

	t.Run("ParseTimeFrame rejects invalid time frames", func(t *testing.T) {
		_, err := ParseTimeFrame("invalid")
		if err == nil {
			t.Error("Expected error for invalid time frame, got none")
		}
	})
}

func TestCandle(t *testing.T) {
	t.Run("Candle.Validate validates candle data", func(t *testing.T) {
		// Valid candle
		c := Candle{
			Timestamp: time.Now().Unix(),
			Open:      100.0,
			High:      110.0,
			Low:       90.0,
			Close:     105.0,
			Volume:    1000.0,
		}
		if err := c.Validate(); err != nil {
			t.Errorf("Expected valid candle, got error: %v", err)
		}

		// Test invalid timestamp
		c.Timestamp = 0
		if err := c.Validate(); err == nil {
			t.Error("Expected error for invalid timestamp, got none")
		}
		c.Timestamp = time.Now().Unix() // Reset for next test

		// Test invalid low/high
		c.Low = 120.0
		if err := c.Validate(); err == nil {
			t.Error("Expected error when low > high, got none")
		}
		c.Low = 90.0 // Reset for next test

		// Test open outside range
		c.Open = 120.0
		if err := c.Validate(); err == nil {
			t.Error("Expected error when open > high, got none")
		}
		c.Open = 100.0 // Reset for next test

		// Test close outside range
		c.Close = 80.0
		if err := c.Validate(); err == nil {
			t.Error("Expected error when close < low, got none")
		}
		c.Close = 105.0 // Reset for next test

		// Test negative volume
		c.Volume = -10.0
		if err := c.Validate(); err == nil {
			t.Error("Expected error for negative volume, got none")
		}
	})

	t.Run("Candle.Time converts timestamp to time.Time", func(t *testing.T) {
		timestamp := int64(1620000000)
		c := Candle{Timestamp: timestamp}
		expected := time.Unix(timestamp, 0)
		if !c.Time().Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, c.Time())
		}
	})

	t.Run("Candle.DateString formats date correctly", func(t *testing.T) {
		// May 3, 2021 UTC
		c := Candle{Timestamp: 1620000000}
		expected := "2021-05-03"
		if c.DateString() != expected {
			t.Errorf("Expected %s, got %s", expected, c.DateString())
		}
	})
}

func TestMetadata(t *testing.T) {
	t.Run("Metadata.Validate validates metadata", func(t *testing.T) {
		// Valid metadata
		m := Metadata{
			Symbol:             "SOL-USD",
			PolygonSymbol:      "X:SOLUSD",
			StartDate:          "2021-01-01",
			EndDate:            "2021-12-31",
			FetchTimestamp:     "2022-01-01 12:00:00",
			DailyCandlesCount:  365,
			HourlyCandlesCount: 8760,
		}
		if err := m.Validate(); err != nil {
			t.Errorf("Expected valid metadata, got error: %v", err)
		}

		// Test empty symbol
		mInvalid := m
		mInvalid.Symbol = ""
		if err := mInvalid.Validate(); err == nil {
			t.Error("Expected error for empty symbol, got none")
		}

		// Test empty polygon symbol
		mInvalid = m
		mInvalid.PolygonSymbol = ""
		if err := mInvalid.Validate(); err == nil {
			t.Error("Expected error for empty polygon symbol, got none")
		}

		// Test invalid date format
		mInvalid = m
		mInvalid.StartDate = "01/01/2021"
		if err := mInvalid.Validate(); err == nil {
			t.Error("Expected error for invalid date format, got none")
		}

		// Test negative candle counts
		mInvalid = m
		mInvalid.DailyCandlesCount = -1
		if err := mInvalid.Validate(); err == nil {
			t.Error("Expected error for negative daily candles count, got none")
		}
	})

	t.Run("Metadata.UpdateFetchTimestamp updates timestamp", func(t *testing.T) {
		m := Metadata{}
		m.UpdateFetchTimestamp()
		if m.FetchTimestamp == "" {
			t.Error("Expected fetch timestamp to be set, got empty string")
		}
	})
}