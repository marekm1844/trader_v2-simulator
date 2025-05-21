package models

import "fmt"

// TimeFrame represents the time interval for price candles
type TimeFrame string

// TimeFrame constants for supported intervals
const (
	TimeFrameDaily  TimeFrame = "daily"
	TimeFrameHourly TimeFrame = "hourly"
)

// IsValid checks if the TimeFrame is one of the supported values
func (tf TimeFrame) IsValid() bool {
	switch tf {
	case TimeFrameDaily, TimeFrameHourly:
		return true
	default:
		return false
	}
}

// Directory returns the directory name for the given timeframe
func (tf TimeFrame) Directory() string {
	return string(tf)
}

// String returns the string representation of the TimeFrame
func (tf TimeFrame) String() string {
	return string(tf)
}

// ParseTimeFrame converts a string to a TimeFrame
func ParseTimeFrame(s string) (TimeFrame, error) {
	tf := TimeFrame(s)
	if !tf.IsValid() {
		return "", fmt.Errorf("invalid timeframe: %s", s)
	}
	return tf, nil
}