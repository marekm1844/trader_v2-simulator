package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Config holds configuration for the simulator
type Config struct {
	DailyInterval  time.Duration // How often to emit daily candles in simulation time
	HourlyInterval time.Duration // How often to emit hourly candles in simulation time
	Symbols        []string      // Trading symbols to include in simulation
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		DailyInterval:  24 * time.Hour,   // Daily candles every 24 hours of sim time
		HourlyInterval: 1 * time.Hour,    // Hourly candles every hour of sim time
		Symbols:        []string{"SOL-USD"},
	}
}

// SimulatorImpl implements the Simulator interface
type SimulatorImpl struct {
	mu             sync.RWMutex
	clock          SimulationClock
	dataProvider   DataProvider
	listeners      map[string]CandleListener
	status         SimulationStatus
	config         Config
	cancelCtx      context.Context
	cancelFunc     context.CancelFunc
	startTime      time.Time
	endTime        time.Time
	lastDailyTime  map[string]time.Time // Last emitted daily candle time by symbol
	lastHourlyTime map[string]time.Time // Last emitted hourly candle time by symbol
}

// NewSimulator creates a new simulator instance
func NewSimulator(clock SimulationClock, dataProvider DataProvider, config Config) *SimulatorImpl {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SimulatorImpl{
		clock:          clock,
		dataProvider:   dataProvider,
		listeners:      make(map[string]CandleListener),
		status:         StatusStopped,
		config:         config,
		cancelCtx:      ctx,
		cancelFunc:     cancel,
		lastDailyTime:  make(map[string]time.Time),
		lastHourlyTime: make(map[string]time.Time),
	}
}

// Start begins the simulation from the specified start time
func (s *SimulatorImpl) Start(ctx context.Context, startTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusRunning {
		return fmt.Errorf("simulator is already running")
	}

	// Find data boundaries for all symbols
	var earliestStart time.Time
	var latestEnd time.Time
	var err error

	for i, symbol := range s.config.Symbols {
		start, end, err := s.dataProvider.GetDataRange(ctx, symbol, TimeFrameDaily)
		if err != nil {
			return fmt.Errorf("failed to get data range for symbol %s: %w", symbol, err)
		}

		if i == 0 || start.Before(earliestStart) {
			earliestStart = start
		}

		if i == 0 || end.After(latestEnd) {
			latestEnd = end
		}
	}

	// Validate start time is within data range
	if startTime.Before(earliestStart) {
		return fmt.Errorf("start time %v is before available data range %v", startTime, earliestStart)
	}

	if startTime.After(latestEnd) {
		return fmt.Errorf("start time %v is after available data range %v", startTime, latestEnd)
	}

	s.startTime = startTime
	s.endTime = latestEnd

	// Initialize candle tracking maps
	for _, symbol := range s.config.Symbols {
		s.lastDailyTime[symbol] = startTime.Add(-24 * time.Hour)  // Ensure first daily candle is emitted
		s.lastHourlyTime[symbol] = startTime.Add(-1 * time.Hour)  // Ensure first hourly candle is emitted
	}

	// Set the clock to the start time
	err = s.clock.SetTime(startTime)
	if err != nil {
		return fmt.Errorf("failed to set clock time: %w", err)
	}

	// Register event handlers for candle emission
	s.registerCandleEvents(ctx)

	// Start the clock
	err = s.clock.Start()
	if err != nil {
		return fmt.Errorf("failed to start clock: %w", err)
	}

	s.status = StatusRunning
	return nil
}

// registerCandleEvents sets up the periodic emission of candles
func (s *SimulatorImpl) registerCandleEvents(ctx context.Context) {
	// Create a goroutine to monitor for candle events
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkCandleEmissions(ctx)
			case <-s.cancelCtx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// checkCandleEmissions checks if it's time to emit any candles
func (s *SimulatorImpl) checkCandleEmissions(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusRunning {
		return
	}

	currentTime := s.clock.GetCurrentTime()

	// Check for daily candles
	for _, symbol := range s.config.Symbols {
		lastDaily := s.lastDailyTime[symbol]
		if currentTime.Sub(lastDaily) >= s.config.DailyInterval {
			// Time to emit a daily candle
			nextCandleTime := lastDaily.Add(s.config.DailyInterval)
			if nextCandleTime.After(s.endTime) {
				continue
			}

			// Get the daily candle data
			candles, err := s.dataProvider.GetCandles(
				ctx, 
				symbol, 
				TimeFrameDaily, 
				nextCandleTime, 
				nextCandleTime.Add(24*time.Hour),
			)
			
			if err == nil && len(candles) > 0 {
				s.emitCandle(candles[0])
				s.lastDailyTime[symbol] = nextCandleTime
			}
		}

		// Check for hourly candles
		lastHourly := s.lastHourlyTime[symbol]
		if currentTime.Sub(lastHourly) >= s.config.HourlyInterval {
			// Time to emit an hourly candle
			nextCandleTime := lastHourly.Add(s.config.HourlyInterval)
			if nextCandleTime.After(s.endTime) {
				continue
			}

			// Get the hourly candle data
			candles, err := s.dataProvider.GetCandles(
				ctx, 
				symbol, 
				TimeFrameHourly, 
				nextCandleTime, 
				nextCandleTime.Add(time.Hour),
			)
			
			if err == nil && len(candles) > 0 {
				s.emitCandle(candles[0])
				s.lastHourlyTime[symbol] = nextCandleTime
			}
		}
	}
}

// emitCandle sends a candle to all registered listeners
func (s *SimulatorImpl) emitCandle(candle Candle) {
	for _, listener := range s.listeners {
		// Send to each listener in a separate goroutine to avoid blocking
		go func(l CandleListener, c Candle) {
			l.OnCandle(c)
		}(listener, candle)
	}
}

// Stop halts the simulation completely
func (s *SimulatorImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusStopped {
		return fmt.Errorf("simulator is already stopped")
	}

	// Cancel any ongoing operations
	s.cancelFunc()

	// Create a new cancel context for future use
	s.cancelCtx, s.cancelFunc = context.WithCancel(context.Background())

	// Stop the clock
	err := s.clock.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop clock: %w", err)
	}

	s.status = StatusStopped
	return nil
}

// Pause temporarily halts the simulation while maintaining state
func (s *SimulatorImpl) Pause(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusRunning {
		return fmt.Errorf("simulator is not running, current status: %s", s.status)
	}

	// Pause the clock
	err := s.clock.Pause()
	if err != nil {
		return fmt.Errorf("failed to pause clock: %w", err)
	}

	s.status = StatusPaused
	return nil
}

// Resume continues a paused simulation
func (s *SimulatorImpl) Resume(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusPaused {
		return fmt.Errorf("simulator is not paused, current status: %s", s.status)
	}

	// Resume the clock
	err := s.clock.Resume()
	if err != nil {
		return fmt.Errorf("failed to resume clock: %w", err)
	}

	s.status = StatusRunning
	return nil
}

// SetSpeed adjusts the simulation speed multiplier
func (s *SimulatorImpl) SetSpeed(speed SimulationSpeed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update the clock speed
	err := s.clock.SetSpeed(speed)
	if err != nil {
		return fmt.Errorf("failed to set clock speed: %w", err)
	}

	return nil
}

// GetSpeed returns the current simulation speed
func (s *SimulatorImpl) GetSpeed() SimulationSpeed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.clock.(*SimulationClockImpl).speed
}

// GetStatus returns the current simulation status
func (s *SimulatorImpl) GetStatus() SimulationStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.status
}

// GetCurrentTime returns the current simulation time
func (s *SimulatorImpl) GetCurrentTime() time.Time {
	return s.clock.GetCurrentTime()
}

// RegisterListener registers a component to receive candle updates
func (s *SimulatorImpl) RegisterListener(listener CandleListener) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := listener.GetListenerID()
	if _, exists := s.listeners[id]; exists {
		return fmt.Errorf("listener with ID %s is already registered", id)
	}

	s.listeners[id] = listener
	return nil
}

// UnregisterListener removes a previously registered listener
func (s *SimulatorImpl) UnregisterListener(listener CandleListener) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := listener.GetListenerID()
	if _, exists := s.listeners[id]; !exists {
		return fmt.Errorf("listener with ID %s is not registered", id)
	}

	delete(s.listeners, id)
	return nil
}

// SkipTo advances the simulation to a specific time
func (s *SimulatorImpl) SkipTo(ctx context.Context, targetTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if targetTime.Before(s.clock.GetCurrentTime()) {
		return fmt.Errorf("cannot skip to a time in the past")
	}

	if targetTime.After(s.endTime) {
		return fmt.Errorf("target time is beyond available data range")
	}

	wasRunning := s.status == StatusRunning

	// Pause the simulation if it's running
	if wasRunning {
		err := s.clock.Pause()
		if err != nil {
			return fmt.Errorf("failed to pause clock: %w", err)
		}
	}

	// Set the clock to the target time
	err := s.clock.SetTime(targetTime)
	if err != nil {
		return fmt.Errorf("failed to set clock time: %w", err)
	}

	// Update last candle times to ensure correct candle emission after skip
	for _, symbol := range s.config.Symbols {
		// Calculate the most recent candle time before the target
		dailyOffset := targetTime.Sub(s.startTime) % s.config.DailyInterval
		hourlyOffset := targetTime.Sub(s.startTime) % s.config.HourlyInterval
		
		s.lastDailyTime[symbol] = targetTime.Add(-dailyOffset)
		s.lastHourlyTime[symbol] = targetTime.Add(-hourlyOffset)
	}

	// Resume the simulation if it was running
	if wasRunning {
		err := s.clock.Resume()
		if err != nil {
			return fmt.Errorf("failed to resume clock: %w", err)
		}
	}

	return nil
}