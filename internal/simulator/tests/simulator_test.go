package tests

import (
	"context"
	"sync"
	"testing"
	"time"
)

// MockClock implements a simple simulation clock for testing
type MockClock struct {
	currentTime time.Time
	speed       float64
	mu          sync.Mutex
	callbacks   map[time.Time][]func()
}

func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{
		currentTime: initialTime,
		speed:       1.0,
		callbacks:   make(map[time.Time][]func()),
	}
}

func (c *MockClock) GetCurrentTime() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentTime
}

func (c *MockClock) Advance(duration time.Duration) {
	c.mu.Lock()
	// Calculate the new time
	newTime := c.currentTime.Add(duration)
	
	// Identify callbacks to execute (copy to avoid race conditions)
	// We need to fire all callbacks that fall within the time window
	callbacksToExecute := make(map[time.Time][]func())
	for t, callbacks := range c.callbacks {
		if !t.After(newTime) && !t.Before(c.currentTime) {
			callbacksToExecute[t] = callbacks
			delete(c.callbacks, t)
		}
	}
	
	// Update the current time
	c.currentTime = newTime
	c.mu.Unlock()
	
	// Execute callbacks outside the lock to prevent deadlocks
	for _, callbacks := range callbacksToExecute {
		for _, callback := range callbacks {
			callback()
		}
	}
}

func (c *MockClock) RegisterCallback(eventTime time.Time, callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.callbacks[eventTime]; !exists {
		c.callbacks[eventTime] = make([]func(), 0)
	}
	c.callbacks[eventTime] = append(c.callbacks[eventTime], callback)
}

// Candle represents a price candle
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// TimeFrame represents the candle interval
type TimeFrame string

const (
	TimeFrameDaily  TimeFrame = "daily"
	TimeFrameHourly TimeFrame = "hourly"
)

// MockDataProvider simulates a data provider for testing
type MockDataProvider struct {
	candles map[string]map[TimeFrame][]Candle
	mu      sync.RWMutex
}

func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{
		candles: make(map[string]map[TimeFrame][]Candle),
	}
}

func (p *MockDataProvider) AddCandles(symbol string, timeFrame TimeFrame, candles []Candle) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.candles[symbol]; !exists {
		p.candles[symbol] = make(map[TimeFrame][]Candle)
	}
	p.candles[symbol][timeFrame] = candles
}

func (p *MockDataProvider) GetCandles(ctx context.Context, symbol string, timeFrame TimeFrame, 
	startTime, endTime time.Time) ([]Candle, error) {
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if _, exists := p.candles[symbol]; !exists {
		return []Candle{}, nil
	}
	
	if _, exists := p.candles[symbol][timeFrame]; !exists {
		return []Candle{}, nil
	}
	
	result := make([]Candle, 0)
	for _, candle := range p.candles[symbol][timeFrame] {
		if (candle.Timestamp.Equal(startTime) || candle.Timestamp.After(startTime)) && 
		   (candle.Timestamp.Before(endTime) || candle.Timestamp.Equal(endTime)) {
			result = append(result, candle)
		}
	}
	
	return result, nil
}

// CandleListener interface for components that receive candle updates
type CandleListener interface {
	OnCandle(candle Candle)
}

// MockListener implements CandleListener for testing
type MockListener struct {
	receivedCandles []Candle
	mu              sync.Mutex
}

func NewMockListener() *MockListener {
	return &MockListener{
		receivedCandles: make([]Candle, 0),
	}
}

func (l *MockListener) OnCandle(candle Candle) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.receivedCandles = append(l.receivedCandles, candle)
}

func (l *MockListener) GetReceivedCandles() []Candle {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Return a copy to avoid race conditions
	result := make([]Candle, len(l.receivedCandles))
	copy(result, l.receivedCandles)
	return result
}

// SimpleSimulator implements a basic simulation for testing
type SimpleSimulator struct {
	clock        *MockClock
	dataProvider *MockDataProvider
	listeners    []CandleListener
	isRunning    bool
	symbol       string
	timeFrame    TimeFrame
	mu           sync.RWMutex
}

func NewSimpleSimulator(clock *MockClock, dataProvider *MockDataProvider) *SimpleSimulator {
	return &SimpleSimulator{
		clock:        clock,
		dataProvider: dataProvider,
		listeners:    make([]CandleListener, 0),
		isRunning:    false,
	}
}

func (s *SimpleSimulator) RegisterListener(listener CandleListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *SimpleSimulator) Start(ctx context.Context, symbol string, timeFrame TimeFrame, startTime time.Time) error {
	s.mu.Lock()
	s.symbol = symbol
	s.timeFrame = timeFrame
	s.isRunning = true
	s.mu.Unlock()
	
	// Determine the appropriate end time based on the current time
	endTime := startTime.Add(48 * time.Hour) // For testing purposes, get 48h of data
	
	// Get candles for the time range
	candles, err := s.dataProvider.GetCandles(ctx, symbol, timeFrame, startTime, endTime)
	if err != nil {
		return err
	}
	
	// Schedule candle emissions - note that we immediately emit candles with matching timestamps
	// This is important for testing, though in a real implementation you might want different behavior
	currentTime := s.clock.GetCurrentTime()
	for _, candle := range candles {
		// Use a copy of the candle in the closure to avoid issues with the loop variable
		c := candle
		if c.Timestamp.Equal(currentTime) {
			// Emit immediately if it's at the current time
			s.emitCandle(c)
		} else if c.Timestamp.After(currentTime) {
			// Schedule for future emission
			s.clock.RegisterCallback(c.Timestamp, func() {
				s.emitCandle(c)
			})
		}
	}
	
	return nil
}

func (s *SimpleSimulator) emitCandle(candle Candle) {
	s.mu.RLock()
	// Make a copy of the listeners slice to avoid holding the lock during callbacks
	listeners := make([]CandleListener, len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.RUnlock()
	
	// Notify all listeners
	for _, listener := range listeners {
		listener.OnCandle(candle)
	}
}

// TestSimulatorCandleEmission tests that candles are emitted correctly
func TestSimulatorCandleEmission(t *testing.T) {
	// Create a context
	ctx := context.Background()
	
	// Create test data
	symbol := "SOL-USD"
	timeFrame := TimeFrameDaily
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Create test candles
	candles := []Candle{
		{
			Symbol:    symbol,
			Timestamp: startTime,  // Jan 1, 2025
			Open:      100.0,
			High:      105.0,
			Low:       98.0,
			Close:     103.0,
			Volume:    1000.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(24 * time.Hour),  // Jan 2, 2025
			Open:      103.0,
			High:      110.0,
			Low:       102.0,
			Close:     108.0,
			Volume:    1200.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(48 * time.Hour),  // Jan 3, 2025
			Open:      108.0,
			High:      112.0,
			Low:       107.0,
			Close:     111.0,
			Volume:    1100.0,
		},
	}
	
	// Create the clock with the same start time
	clock := NewMockClock(startTime)
	
	// Create the data provider and add test data
	dataProvider := NewMockDataProvider()
	dataProvider.AddCandles(symbol, timeFrame, candles)
	
	// Create the simulator
	simulator := NewSimpleSimulator(clock, dataProvider)
	
	// Create a listener to receive candles
	listener := NewMockListener()
	simulator.RegisterListener(listener)
	
	// Start the simulator
	err := simulator.Start(ctx, symbol, timeFrame, startTime)
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	
	// First check - we should have one candle immediately (the one at the start time)
	receivedCandles := listener.GetReceivedCandles()
	if len(receivedCandles) != 1 {
		t.Fatalf("Expected 1 candle immediately, got %d", len(receivedCandles))
	}
	
	if receivedCandles[0].Timestamp != startTime {
		t.Errorf("Expected candle timestamp %v, got %v", startTime, receivedCandles[0].Timestamp)
	}
	
	// Advance to next day to get the second candle
	clock.Advance(24 * time.Hour)
	
	// Check for second candle
	receivedCandles = listener.GetReceivedCandles()
	if len(receivedCandles) != 2 {
		t.Fatalf("Expected 2 candles after 24h, got %d", len(receivedCandles))
	}
	
	if receivedCandles[1].Timestamp != startTime.Add(24*time.Hour) {
		t.Errorf("Expected second candle timestamp %v, got %v", 
			startTime.Add(24*time.Hour), receivedCandles[1].Timestamp)
	}
	
	// Advance to get the third candle
	clock.Advance(24 * time.Hour)
	
	// Check for all three candles
	receivedCandles = listener.GetReceivedCandles()
	if len(receivedCandles) != 3 {
		t.Fatalf("Expected 3 candles after 48h, got %d", len(receivedCandles))
	}
	
	if receivedCandles[2].Timestamp != startTime.Add(48*time.Hour) {
		t.Errorf("Expected third candle timestamp %v, got %v", 
			startTime.Add(48*time.Hour), receivedCandles[2].Timestamp)
	}
	
	// Verify the candle data
	for i, candle := range candles {
		if receivedCandles[i].Open != candle.Open ||
			receivedCandles[i].High != candle.High ||
			receivedCandles[i].Low != candle.Low ||
			receivedCandles[i].Close != candle.Close ||
			receivedCandles[i].Volume != candle.Volume {
			t.Errorf("Candle %d data mismatch", i)
		}
	}
}

// TestMultipleTimeframes tests handling of multiple timeframes
func TestMultipleTimeframes(t *testing.T) {
	// Create a context
	ctx := context.Background()
	
	// Create test data
	symbol := "SOL-USD"
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Create daily candles
	dailyCandles := []Candle{
		{
			Symbol:    symbol,
			Timestamp: startTime,
			Open:      100.0,
			High:      105.0,
			Low:       98.0,
			Close:     103.0,
			Volume:    1000.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(24 * time.Hour),
			Open:      103.0,
			High:      110.0,
			Low:       102.0,
			Close:     108.0,
			Volume:    1200.0,
		},
	}
	
	// Create hourly candles
	hourlyCandles := []Candle{
		{
			Symbol:    symbol,
			Timestamp: startTime,  // Hour 0
			Open:      100.0,
			High:      101.0,
			Low:       99.5,
			Close:     100.5,
			Volume:    200.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(1 * time.Hour),  // Hour 1
			Open:      100.5,
			High:      102.0,
			Low:       100.0,
			Close:     101.5,
			Volume:    210.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(2 * time.Hour),  // Hour 2
			Open:      101.5,
			High:      103.0,
			Low:       101.0,
			Close:     102.5,
			Volume:    220.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(24 * time.Hour),  // Day 2, Hour 0
			Open:      103.0,
			High:      104.0,
			Low:       102.5,
			Close:     103.5,
			Volume:    230.0,
		},
	}
	
	// Create the clock with the start time
	clock := NewMockClock(startTime)
	
	// Create the data provider and add test data
	dataProvider := NewMockDataProvider()
	dataProvider.AddCandles(symbol, TimeFrameDaily, dailyCandles)
	dataProvider.AddCandles(symbol, TimeFrameHourly, hourlyCandles)
	
	// Create simulators for each timeframe
	dailySimulator := NewSimpleSimulator(clock, dataProvider)
	hourlySimulator := NewSimpleSimulator(clock, dataProvider)
	
	// Create listeners
	dailyListener := NewMockListener()
	hourlyListener := NewMockListener()
	
	dailySimulator.RegisterListener(dailyListener)
	hourlySimulator.RegisterListener(hourlyListener)
	
	// Start the simulators
	err := dailySimulator.Start(ctx, symbol, TimeFrameDaily, startTime)
	if err != nil {
		t.Fatalf("Failed to start daily simulator: %v", err)
	}
	
	err = hourlySimulator.Start(ctx, symbol, TimeFrameHourly, startTime)
	if err != nil {
		t.Fatalf("Failed to start hourly simulator: %v", err)
	}
	
	// First check - we should have candles at the start time for both timeframes
	dailyCandles = dailyListener.GetReceivedCandles()
	hourlyCandles = hourlyListener.GetReceivedCandles()
	
	if len(dailyCandles) != 1 {
		t.Errorf("Expected 1 daily candle at start, got %d", len(dailyCandles))
	}
	
	if len(hourlyCandles) != 1 {
		t.Errorf("Expected 1 hourly candle at start, got %d", len(hourlyCandles))
	}
	
	// Advance one hour to get the next hourly candle
	clock.Advance(1 * time.Hour)
	
	hourlyCandles = hourlyListener.GetReceivedCandles()
	if len(hourlyCandles) != 2 {
		t.Errorf("Expected 2 hourly candles after 1h, got %d", len(hourlyCandles))
	}
	
	// Advance another hour to get the third hourly candle
	clock.Advance(1 * time.Hour)
	
	hourlyCandles = hourlyListener.GetReceivedCandles()
	if len(hourlyCandles) != 3 {
		t.Errorf("Expected 3 hourly candles after 2h, got %d", len(hourlyCandles))
	}
	
	// Advance to next day to get the second daily candle and fourth hourly candle
	clock.Advance(22 * time.Hour) // Takes us to exactly 24 hours from start
	
	dailyCandles = dailyListener.GetReceivedCandles()
	hourlyCandles = hourlyListener.GetReceivedCandles()
	
	if len(dailyCandles) != 2 {
		t.Errorf("Expected 2 daily candles after 24h, got %d", len(dailyCandles))
	}
	
	if len(hourlyCandles) != 4 {
		t.Errorf("Expected 4 hourly candles after 24h, got %d", len(hourlyCandles))
	}
}