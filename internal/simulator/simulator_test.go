package simulator

import (
	"context"
	"sync"
	"testing"
	"time"
)

// MockClock implements SimulationClock for testing
type MockClock struct {
	mu          sync.RWMutex
	currentTime time.Time
	speed       SimulationSpeed
	status      SimulationStatus
	timedEvents []TimedEvent
}

func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{
		currentTime: initialTime,
		speed:       SpeedRealtime,
		status:      StatusStopped,
		timedEvents: make([]TimedEvent, 0),
	}
}

func (m *MockClock) GetCurrentTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTime
}

func (m *MockClock) Advance(duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = m.currentTime.Add(duration)
	m.processTimedEvents()
	return nil
}

func (m *MockClock) SetTime(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = t
	m.processTimedEvents()
	return nil
}

func (m *MockClock) SetSpeed(speed SimulationSpeed) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.speed = speed
	return nil
}

func (m *MockClock) RegisterTimedEvent(eventTime time.Time, eventFn func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timedEvents = append(m.timedEvents, TimedEvent{
		eventTime: eventTime,
		eventFn:   eventFn,
	})
	return nil
}

func (m *MockClock) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusRunning
	return nil
}

func (m *MockClock) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusStopped
	return nil
}

func (m *MockClock) Pause() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusPaused
	return nil
}

func (m *MockClock) Resume() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusRunning
	return nil
}

func (m *MockClock) processTimedEvents() {
	pendingEvents := make([]TimedEvent, 0, len(m.timedEvents))
	
	for _, event := range m.timedEvents {
		if !event.eventTime.After(m.currentTime) {
			go event.eventFn()
		} else {
			pendingEvents = append(pendingEvents, event)
		}
	}
	
	m.timedEvents = pendingEvents
}

// MockDataProvider implements DataProvider for testing
type MockDataProvider struct {
	candles     map[string]map[TimeFrame][]Candle
	dataRange   map[string]map[TimeFrame]struct{ start, end time.Time }
	symbols     []string
}

func NewMockDataProvider(symbols []string) *MockDataProvider {
	candles := make(map[string]map[TimeFrame][]Candle)
	dataRange := make(map[string]map[TimeFrame]struct{ start, end time.Time })
	
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	
	for _, symbol := range symbols {
		candles[symbol] = make(map[TimeFrame][]Candle)
		dataRange[symbol] = make(map[TimeFrame]struct{ start, end time.Time })
		
		// Create daily candles
		dailyCandles := make([]Candle, 0)
		for d := 0; d < 31; d++ {
			timestamp := startDate.AddDate(0, 0, d)
			dailyCandles = append(dailyCandles, Candle{
				Timestamp: timestamp,
				Symbol:    symbol,
				Open:      100.0 + float64(d),
				High:      101.0 + float64(d),
				Low:       99.0 + float64(d),
				Close:     100.5 + float64(d),
				Volume:    1000.0 + float64(d*100),
			})
		}
		candles[symbol][TimeFrameDaily] = dailyCandles
		dataRange[symbol][TimeFrameDaily] = struct{ start, end time.Time }{startDate, endDate}
		
		// Create hourly candles
		hourlyCandles := make([]Candle, 0)
		for d := 0; d < 31; d++ {
			dayBase := startDate.AddDate(0, 0, d)
			for h := 0; h < 24; h++ {
				timestamp := dayBase.Add(time.Duration(h) * time.Hour)
				hourlyCandles = append(hourlyCandles, Candle{
					Timestamp: timestamp,
					Symbol:    symbol,
					Open:      100.0 + float64(d) + (float64(h) * 0.1),
					High:      101.0 + float64(d) + (float64(h) * 0.1),
					Low:       99.0 + float64(d) + (float64(h) * 0.1),
					Close:     100.5 + float64(d) + (float64(h) * 0.1),
					Volume:    100.0 + float64(h*10),
				})
			}
		}
		candles[symbol][TimeFrameHourly] = hourlyCandles
		dataRange[symbol][TimeFrameHourly] = struct{ start, end time.Time }{startDate, endDate.Add(23 * time.Hour)}
	}
	
	return &MockDataProvider{
		candles:   candles,
		dataRange: dataRange,
		symbols:   symbols,
	}
}

func (m *MockDataProvider) GetCandles(ctx context.Context, symbol string, timeframe TimeFrame, 
	startTime, endTime time.Time) ([]Candle, error) {
	
	allCandles := m.candles[symbol][timeframe]
	result := make([]Candle, 0)
	
	for _, candle := range allCandles {
		if (candle.Timestamp.Equal(startTime) || candle.Timestamp.After(startTime)) &&
			candle.Timestamp.Before(endTime) {
			result = append(result, candle)
		}
	}
	
	return result, nil
}

func (m *MockDataProvider) GetAvailableSymbols(ctx context.Context) ([]string, error) {
	return m.symbols, nil
}

func (m *MockDataProvider) GetDataRange(ctx context.Context, symbol string, timeframe TimeFrame) (startTime, endTime time.Time, err error) {
	r := m.dataRange[symbol][timeframe]
	return r.start, r.end, nil
}

func (m *MockDataProvider) Reload(ctx context.Context) error {
	return nil
}

// MockListener implements CandleListener for testing
type MockListener struct {
	id         string
	candles    []Candle
	candleChan chan Candle
	mu         sync.RWMutex
}

func NewMockListener(id string) *MockListener {
	return &MockListener{
		id:         id,
		candles:    make([]Candle, 0),
		candleChan: make(chan Candle, 100),
	}
}

func (m *MockListener) OnCandle(candle Candle) {
	m.mu.Lock()
	m.candles = append(m.candles, candle)
	m.mu.Unlock()
	
	// Non-blocking send to channel
	select {
	case m.candleChan <- candle:
	default:
		// Channel is full, discard
	}
}

func (m *MockListener) GetListenerID() string {
	return m.id
}

func (m *MockListener) GetReceived() []Candle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]Candle, len(m.candles))
	copy(result, m.candles)
	return result
}

// Tests

func TestSimulatorStart(t *testing.T) {
	ctx := context.Background()
	
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	symbols := []string{"SOL-USD"}
	
	mockClock := NewMockClock(startDate)
	mockDataProvider := NewMockDataProvider(symbols)
	
	config := DefaultConfig()
	config.Symbols = symbols
	
	simulator := NewSimulator(mockClock, mockDataProvider, config)
	
	// Register a listener
	listener := NewMockListener("test-listener")
	err := simulator.RegisterListener(listener)
	if err != nil {
		t.Fatalf("Failed to register listener: %v", err)
	}
	
	// Start the simulator
	err = simulator.Start(ctx, startDate)
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	
	// Check status
	if simulator.GetStatus() != StatusRunning {
		t.Errorf("Expected status to be running, got %s", simulator.GetStatus())
	}
	
	// Advance time to trigger candle emission
	mockClock.Advance(1 * time.Hour)
	
	// Verify we received the expected hourly candle
	select {
	case candle := <-listener.candleChan:
		if candle.Symbol != "SOL-USD" {
			t.Errorf("Expected symbol SOL-USD, got %s", candle.Symbol)
		}
		if !candle.Timestamp.Equal(startDate) {
			t.Errorf("Expected timestamp %v, got %v", startDate, candle.Timestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for hourly candle")
	}
	
	// Advance to trigger daily candle
	mockClock.Advance(23 * time.Hour)
	
	// Verify we received the expected daily candle
	select {
	case candle := <-listener.candleChan:
		if candle.Symbol != "SOL-USD" {
			t.Errorf("Expected symbol SOL-USD, got %s", candle.Symbol)
		}
		if !candle.Timestamp.Equal(startDate) {
			t.Errorf("Expected timestamp %v, got %v", startDate, candle.Timestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for daily candle")
	}
	
	// Stop the simulator
	err = simulator.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop simulator: %v", err)
	}
	
	// Check status
	if simulator.GetStatus() != StatusStopped {
		t.Errorf("Expected status to be stopped, got %s", simulator.GetStatus())
	}
}

func TestSimulatorPauseResume(t *testing.T) {
	ctx := context.Background()
	
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	symbols := []string{"SOL-USD"}
	
	mockClock := NewMockClock(startDate)
	mockDataProvider := NewMockDataProvider(symbols)
	
	config := DefaultConfig()
	config.Symbols = symbols
	
	simulator := NewSimulator(mockClock, mockDataProvider, config)
	
	// Start the simulator
	err := simulator.Start(ctx, startDate)
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	
	// Pause the simulator
	err = simulator.Pause(ctx)
	if err != nil {
		t.Fatalf("Failed to pause simulator: %v", err)
	}
	
	// Check status
	if simulator.GetStatus() != StatusPaused {
		t.Errorf("Expected status to be paused, got %s", simulator.GetStatus())
	}
	
	// Resume the simulator
	err = simulator.Resume(ctx)
	if err != nil {
		t.Fatalf("Failed to resume simulator: %v", err)
	}
	
	// Check status
	if simulator.GetStatus() != StatusRunning {
		t.Errorf("Expected status to be running, got %s", simulator.GetStatus())
	}
	
	// Stop the simulator
	err = simulator.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop simulator: %v", err)
	}
}

func TestSimulatorSkipTo(t *testing.T) {
	ctx := context.Background()
	
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	symbols := []string{"SOL-USD"}
	
	mockClock := NewMockClock(startDate)
	mockDataProvider := NewMockDataProvider(symbols)
	
	config := DefaultConfig()
	config.Symbols = symbols
	
	simulator := NewSimulator(mockClock, mockDataProvider, config)
	
	// Register a listener
	listener := NewMockListener("test-listener")
	err := simulator.RegisterListener(listener)
	if err != nil {
		t.Fatalf("Failed to register listener: %v", err)
	}
	
	// Start the simulator
	err = simulator.Start(ctx, startDate)
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	
	// Skip to a future date
	targetDate := startDate.AddDate(0, 0, 5)
	err = simulator.SkipTo(ctx, targetDate)
	if err != nil {
		t.Fatalf("Failed to skip to target date: %v", err)
	}
	
	// Check current time
	if !simulator.GetCurrentTime().Equal(targetDate) {
		t.Errorf("Expected current time to be %v, got %v", targetDate, simulator.GetCurrentTime())
	}
	
	// Advance time to trigger candle emission after skip
	mockClock.Advance(1 * time.Hour)
	
	// Verify we start receiving candles from the new time
	select {
	case candle := <-listener.candleChan:
		if candle.Timestamp.Before(targetDate) {
			t.Errorf("Received candle from before skip time: candle=%v, skip=%v", 
				candle.Timestamp, targetDate)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for candle after skip")
	}
	
	// Stop the simulator
	err = simulator.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop simulator: %v", err)
	}
}

func TestSimulatorSpeedChanges(t *testing.T) {
	ctx := context.Background()
	
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	symbols := []string{"SOL-USD"}
	
	mockClock := NewMockClock(startDate)
	mockDataProvider := NewMockDataProvider(symbols)
	
	config := DefaultConfig()
	config.Symbols = symbols
	
	simulator := NewSimulator(mockClock, mockDataProvider, config)
	
	// Start the simulator
	err := simulator.Start(ctx, startDate)
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	
	// Change speed
	err = simulator.SetSpeed(SpeedFast)
	if err != nil {
		t.Fatalf("Failed to set speed: %v", err)
	}
	
	// Check speed
	if simulator.GetSpeed() != SpeedFast {
		t.Errorf("Expected speed to be %v, got %v", SpeedFast, simulator.GetSpeed())
	}
	
	// Stop the simulator
	err = simulator.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop simulator: %v", err)
	}
}