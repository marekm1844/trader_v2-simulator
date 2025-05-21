package simulator

import (
	"context"
	"time"
)

// SimulationSpeed represents the speed at which the simulation runs
type SimulationSpeed float64

const (
	SpeedRealtime SimulationSpeed = 1.0
	SpeedFast     SimulationSpeed = 5.0
	SpeedSlow     SimulationSpeed = 0.5
)

// TimeFrame represents the candle interval
type TimeFrame string

const (
	TimeFrameDaily  TimeFrame = "daily"
	TimeFrameHourly TimeFrame = "hourly"
)

// SimulationStatus represents the current state of the simulation
type SimulationStatus string

const (
	StatusStopped  SimulationStatus = "stopped"
	StatusRunning  SimulationStatus = "running"
	StatusPaused   SimulationStatus = "paused"
)

// Candle represents a price candle with OHLCV data
type Candle struct {
	Timestamp time.Time
	Symbol    string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Simulator is the main interface for controlling the simulation
type Simulator interface {
	// Start begins the simulation from the specified start time
	Start(ctx context.Context, startTime time.Time) error
	
	// Stop halts the simulation completely
	Stop(ctx context.Context) error
	
	// Pause temporarily halts the simulation while maintaining state
	Pause(ctx context.Context) error
	
	// Resume continues a paused simulation
	Resume(ctx context.Context) error
	
	// SetSpeed adjusts the simulation speed multiplier
	SetSpeed(speed SimulationSpeed) error
	
	// GetSpeed returns the current simulation speed
	GetSpeed() SimulationSpeed
	
	// GetStatus returns the current simulation status
	GetStatus() SimulationStatus
	
	// GetCurrentTime returns the current simulation time
	GetCurrentTime() time.Time
	
	// RegisterListener registers a component to receive candle updates
	RegisterListener(listener CandleListener) error
	
	// UnregisterListener removes a previously registered listener
	UnregisterListener(listener CandleListener) error
	
	// SkipTo advances the simulation to a specific time
	SkipTo(ctx context.Context, targetTime time.Time) error
}

// DataProvider defines the interface for accessing historical price data
type DataProvider interface {
	// GetCandles fetches candles for a specific symbol, timeframe, and date range
	GetCandles(ctx context.Context, symbol string, timeframe TimeFrame, 
		startTime, endTime time.Time) ([]Candle, error)
	
	// GetAvailableSymbols returns all available trading symbols
	GetAvailableSymbols(ctx context.Context) ([]string, error)
	
	// GetDataRange returns the available data range for a symbol and timeframe
	GetDataRange(ctx context.Context, symbol string, timeframe TimeFrame) (startTime, endTime time.Time, err error)
	
	// Reload refreshes data from the underlying source
	Reload(ctx context.Context) error
}

// SimulationClock manages the progression of time in the simulation
type SimulationClock interface {
	// GetCurrentTime returns the current simulation time
	GetCurrentTime() time.Time
	
	// Advance moves the clock forward by the specified duration
	Advance(duration time.Duration) error
	
	// SetTime sets the clock to a specific time
	SetTime(time time.Time) error
	
	// SetSpeed adjusts how quickly simulation time advances compared to real time
	SetSpeed(speed SimulationSpeed) error
	
	// RegisterTimedEvent schedules a function to be called at a specific simulation time
	RegisterTimedEvent(eventTime time.Time, eventFn func()) error
	
	// Start begins the clock's progression
	Start() error
	
	// Stop halts the clock's progression
	Stop() error
	
	// Pause temporarily halts the clock
	Pause() error
	
	// Resume continues a paused clock
	Resume() error
}

// CandleListener is the interface for components that consume candle updates
type CandleListener interface {
	// OnCandle is called when a new candle is processed in the simulation
	OnCandle(candle Candle)
	
	// GetListenerID returns a unique identifier for this listener
	GetListenerID() string
}