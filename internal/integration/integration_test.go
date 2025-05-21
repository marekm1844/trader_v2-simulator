package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// MockDataProvider provides historical price data for testing
type MockDataProvider struct {
	candles map[string]map[string][]Candle
	mu      sync.RWMutex
}

func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{
		candles: make(map[string]map[string][]Candle),
	}
}

func (p *MockDataProvider) AddCandles(symbol string, timeFrame string, candles []Candle) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.candles[symbol]; !exists {
		p.candles[symbol] = make(map[string][]Candle)
	}
	p.candles[symbol][timeFrame] = candles
}

func (p *MockDataProvider) GetCandles(ctx context.Context, symbol string, timeFrame string, 
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

func (p *MockDataProvider) GetAvailableSymbols(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	symbols := make([]string, 0, len(p.candles))
	for symbol := range p.candles {
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func (p *MockDataProvider) GetDataRange(ctx context.Context, symbol string, timeFrame string) (time.Time, time.Time, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, exists := p.candles[symbol]; !exists {
		return time.Time{}, time.Time{}, fmt.Errorf("no data for symbol %s", symbol)
	}

	if _, exists := p.candles[symbol][timeFrame]; !exists {
		return time.Time{}, time.Time{}, fmt.Errorf("no data for timeframe %s", timeFrame)
	}

	candles := p.candles[symbol][timeFrame]
	if len(candles) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no candles for %s/%s", symbol, timeFrame)
	}

	// Find min and max timestamps
	minTime := candles[0].Timestamp
	maxTime := candles[0].Timestamp

	for _, candle := range candles {
		if candle.Timestamp.Before(minTime) {
			minTime = candle.Timestamp
		}
		if candle.Timestamp.After(maxTime) {
			maxTime = candle.Timestamp
		}
	}

	return minTime, maxTime, nil
}

// Candle represents a price candle
type Candle struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// SimulationClock controls the progression of time in the simulation
type SimulationClock struct {
	mu          sync.Mutex
	currentTime time.Time
	speed       float64
	isPaused    bool
	tickSize    time.Duration
	ticker      *time.Ticker
	stopCh      chan struct{}
	callbacks   map[time.Time][]func()
}

func NewSimulationClock(initialTime time.Time, speed float64, tickSize time.Duration) *SimulationClock {
	return &SimulationClock{
		currentTime: initialTime,
		speed:       speed,
		isPaused:    true,
		tickSize:    tickSize,
		callbacks:   make(map[time.Time][]func()),
		stopCh:      make(chan struct{}),
	}
}

func (c *SimulationClock) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isPaused {
		return fmt.Errorf("clock already running")
	}

	c.isPaused = false
	c.ticker = time.NewTicker(c.tickSize)

	go func() {
		for {
			select {
			case <-c.ticker.C:
				if !c.IsPaused() {
					// Calculate how much simulated time has passed
					simulatedDuration := time.Duration(float64(c.tickSize) * c.speed)
					c.advanceTime(simulatedDuration)
				}
			case <-c.stopCh:
				return
			}
		}
	}()

	return nil
}

func (c *SimulationClock) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ticker != nil {
		c.ticker.Stop()
	}
	
	// Signal the clock goroutine to stop if it's running
	select {
	case c.stopCh <- struct{}{}:
	default:
	}

	c.isPaused = true
	return nil
}

func (c *SimulationClock) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isPaused = true
	return nil
}

func (c *SimulationClock) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isPaused = false
	return nil
}

func (c *SimulationClock) IsPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isPaused
}

func (c *SimulationClock) GetCurrentTime() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentTime
}

func (c *SimulationClock) SetSpeed(speed float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if speed <= 0 {
		return fmt.Errorf("speed must be positive")
	}
	
	c.speed = speed
	return nil
}

func (c *SimulationClock) RegisterCallback(eventTime time.Time, callback func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.callbacks[eventTime]; !exists {
		c.callbacks[eventTime] = make([]func(), 0)
	}
	
	c.callbacks[eventTime] = append(c.callbacks[eventTime], callback)
	return nil
}

func (c *SimulationClock) advanceTime(duration time.Duration) {
	c.mu.Lock()
	
	// Calculate the new time
	newTime := c.currentTime.Add(duration)
	
	// Find callbacks to execute
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
	
	// Execute callbacks outside of the lock
	for _, callbacks := range callbacksToExecute {
		for _, callback := range callbacks {
			callback()
		}
	}
}

// Simulator controls the trading simulation
type Simulator struct {
	mu            sync.RWMutex
	clock         *SimulationClock
	dataProvider  *MockDataProvider
	listeners     []CandleListener
	status        string
	symbol        string
	dailyInterval time.Duration
	hourlyInterval time.Duration
}

const (
	StatusStopped  = "stopped"
	StatusRunning  = "running"
	StatusPaused   = "paused"
)

type CandleListener interface {
	OnCandle(candle Candle)
}

func NewSimulator(clock *SimulationClock, dataProvider *MockDataProvider) *Simulator {
	return &Simulator{
		clock:         clock,
		dataProvider:  dataProvider,
		listeners:     make([]CandleListener, 0),
		status:        StatusStopped,
		dailyInterval: 5 * time.Second,  // Daily candles every 5 seconds
		hourlyInterval: 1 * time.Second, // Hourly candles every 1 second
	}
}

func (s *Simulator) RegisterListener(listener CandleListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *Simulator) UnregisterListener(listener CandleListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find and remove the listener
	for i, l := range s.listeners {
		if l == listener {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}

func (s *Simulator) Start(ctx context.Context, symbol string, startTime time.Time) error {
	s.mu.Lock()
	if s.status != StatusStopped {
		s.mu.Unlock()
		return fmt.Errorf("simulator already running")
	}
	
	s.symbol = symbol
	s.status = StatusRunning
	s.mu.Unlock()
	
	// Start the clock
	err := s.clock.Start()
	if err != nil {
		return err
	}
	
	// Load daily candles
	dailyEndTime := startTime.Add(30 * 24 * time.Hour) // Get 30 days of data
	dailyCandles, err := s.dataProvider.GetCandles(ctx, symbol, "daily", startTime, dailyEndTime)
	if err != nil {
		return err
	}
	
	// Load hourly candles
	hourlyEndTime := startTime.Add(7 * 24 * time.Hour) // Get 7 days of hourly data
	hourlyCandles, err := s.dataProvider.GetCandles(ctx, symbol, "hourly", startTime, hourlyEndTime)
	if err != nil {
		return err
	}
	
	// Schedule daily candles
	var lastDailyTime time.Time
	if len(dailyCandles) > 0 {
		lastDailyTime = dailyCandles[0].Timestamp
		
		// First emit the candle at the start time
		dailyCandle := dailyCandles[0]
		s.emitCandle(dailyCandle)
		
		// Schedule the rest with the specified interval
		nextEmitTime := s.clock.GetCurrentTime().Add(s.dailyInterval)
		for i := 1; i < len(dailyCandles); i++ {
			dailyCandle := dailyCandles[i]
			lastDailyTime = dailyCandle.Timestamp
			
			// Use a closure to capture the current value of dailyCandle
			c := dailyCandle
			s.clock.RegisterCallback(nextEmitTime, func() {
				s.emitCandle(c)
			})
			
			nextEmitTime = nextEmitTime.Add(s.dailyInterval)
		}
	}
	
	// Schedule hourly candles
	var lastHourlyTime time.Time
	if len(hourlyCandles) > 0 {
		lastHourlyTime = hourlyCandles[0].Timestamp
		
		// First emit the candle at the start time
		hourlyCandle := hourlyCandles[0]
		s.emitCandle(hourlyCandle)
		
		// Schedule the rest with the specified interval
		nextEmitTime := s.clock.GetCurrentTime().Add(s.hourlyInterval)
		for i := 1; i < len(hourlyCandles); i++ {
			hourlyCandle := hourlyCandles[i]
			lastHourlyTime = hourlyCandle.Timestamp
			
			// Use a closure to capture the current value of hourlyCandle
			c := hourlyCandle
			s.clock.RegisterCallback(nextEmitTime, func() {
				s.emitCandle(c)
			})
			
			nextEmitTime = nextEmitTime.Add(s.hourlyInterval)
		}
	}
	
	log.Printf("Simulator started for %s with daily candles every %v and hourly candles every %v", 
		symbol, s.dailyInterval, s.hourlyInterval)
	log.Printf("Loaded %d daily candles from %v to %v", 
		len(dailyCandles), startTime, lastDailyTime)
	log.Printf("Loaded %d hourly candles from %v to %v", 
		len(hourlyCandles), startTime, lastHourlyTime)
	
	return nil
}

func (s *Simulator) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.status = StatusStopped
	return s.clock.Stop()
}

func (s *Simulator) Pause(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.status != StatusRunning {
		return fmt.Errorf("simulator not running")
	}
	
	s.status = StatusPaused
	return s.clock.Pause()
}

func (s *Simulator) Resume(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.status != StatusPaused {
		return fmt.Errorf("simulator not paused")
	}
	
	s.status = StatusRunning
	return s.clock.Resume()
}

func (s *Simulator) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Simulator) emitCandle(candle Candle) {
	s.mu.RLock()
	listeners := make([]CandleListener, len(s.listeners))
	copy(listeners, s.listeners)
	s.mu.RUnlock()
	
	// Notify all listeners
	for _, listener := range listeners {
		listener.OnCandle(candle)
	}
}

// WebSocketServer handles WebSocket connections
type WebSocketServer struct {
	simulator     *Simulator
	clients       map[*websocket.Conn]bool
	upgrader      websocket.Upgrader
	clientsMu     sync.Mutex
}

func NewWebSocketServer(simulator *Simulator) *WebSocketServer {
	return &WebSocketServer{
		simulator: simulator,
		clients:   make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections for testing
			},
		},
	}
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	
	// Register the client
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()
	
	log.Printf("Client connected: %s", conn.RemoteAddr().String())
	
	// Create a cleanup function for when the connection closes
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
		conn.Close()
		log.Printf("Client disconnected: %s", conn.RemoteAddr().String())
	}()
	
	// Read messages from the client
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}
		
		// Handle the message
		s.handleMessage(conn, messageType, message)
	}
}

func (s *WebSocketServer) handleMessage(conn *websocket.Conn, messageType int, message []byte) {
	// Parse the message
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("Error parsing message: %v", err)
		return
	}
	
	// Check message type
	msgType, ok := data["type"].(string)
	if !ok {
		log.Printf("Invalid message format: missing type field")
		return
	}
	
	// Handle different message types
	switch msgType {
	case "start":
		symbol, ok := data["symbol"].(string)
		if !ok {
			log.Printf("Invalid start message: missing symbol")
			return
		}
		
		// Get start time (default to current clock time if not provided)
		var startTime time.Time
		startTimeStr, ok := data["start_time"].(string)
		if ok {
			var err error
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				log.Printf("Invalid start time format: %v", err)
				return
			}
		} else {
			startTime = s.simulator.clock.GetCurrentTime()
		}
		
		// Start the simulator
		ctx := context.Background()
		err := s.simulator.Start(ctx, symbol, startTime)
		if err != nil {
			log.Printf("Error starting simulator: %v", err)
			return
		}
		
		// Send confirmation
		response := map[string]interface{}{
			"type":   "status",
			"status": "running",
		}
		conn.WriteJSON(response)
		
	case "stop":
		ctx := context.Background()
		err := s.simulator.Stop(ctx)
		if err != nil {
			log.Printf("Error stopping simulator: %v", err)
			return
		}
		
		// Send confirmation
		response := map[string]interface{}{
			"type":   "status",
			"status": "stopped",
		}
		conn.WriteJSON(response)
		
	case "pause":
		ctx := context.Background()
		err := s.simulator.Pause(ctx)
		if err != nil {
			log.Printf("Error pausing simulator: %v", err)
			return
		}
		
		// Send confirmation
		response := map[string]interface{}{
			"type":   "status",
			"status": "paused",
		}
		conn.WriteJSON(response)
		
	case "resume":
		ctx := context.Background()
		err := s.simulator.Resume(ctx)
		if err != nil {
			log.Printf("Error resuming simulator: %v", err)
			return
		}
		
		// Send confirmation
		response := map[string]interface{}{
			"type":   "status",
			"status": "running",
		}
		conn.WriteJSON(response)
		
	default:
		log.Printf("Unknown message type: %s", msgType)
	}
}

// Implement CandleListener interface to broadcast candles
func (s *WebSocketServer) OnCandle(candle Candle) {
	// Create message
	message := map[string]interface{}{
		"type":   "candle",
		"symbol": candle.Symbol,
		"candle": candle,
	}
	
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error serializing candle message: %v", err)
		return
	}
	
	// Log the candle for testing
	var timeframe string
	if strings.HasSuffix(candle.Timestamp.Format(time.RFC3339), "T00:00:00Z") {
		timeframe = "daily"
	} else {
		timeframe = "hourly"
	}
	
	log.Printf("CANDLE [%s] %s: O: %.2f, H: %.2f, L: %.2f, C: %.2f, V: %.2f", 
		timeframe, candle.Timestamp.Format(time.RFC3339), 
		candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
	
	// Broadcast to all clients
	s.clientsMu.Lock()
	for conn := range s.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			// We don't close the connection here to avoid modifying the map during iteration
		}
	}
	s.clientsMu.Unlock()
}

// WebSocket Client for testing
type WebSocketClient struct {
	conn        *websocket.Conn
	receivedMsg chan []byte
	done        chan struct{}
}

func NewWebSocketClient(url string) (*WebSocketClient, error) {
	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	
	client := &WebSocketClient{
		conn:        conn,
		receivedMsg: make(chan []byte, 100),
		done:        make(chan struct{}),
	}
	
	// Start reading messages
	go func() {
		defer close(client.receivedMsg)
		
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("Client read error: %v", err)
				}
				break
			}
			
			// Send message to channel
			select {
			case client.receivedMsg <- message:
			case <-client.done:
				return
			}
		}
	}()
	
	return client, nil
}

func (c *WebSocketClient) Close() {
	close(c.done)
	c.conn.Close()
}

func (c *WebSocketClient) SendJSON(data interface{}) error {
	return c.conn.WriteJSON(data)
}

func (c *WebSocketClient) ReceiveMessage(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-c.receivedMsg:
		return msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for message")
	}
}

// Integration test using WebSockets
func TestSimulatorWithWebSocket(t *testing.T) {
	// Create data provider with realistic SOL-USD data
	dataProvider := NewMockDataProvider()
	
	// Generate test data
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
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(48 * time.Hour),
			Open:      108.0,
			High:      112.0,
			Low:       107.0,
			Close:     111.0,
			Volume:    1100.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(72 * time.Hour),
			Open:      111.0,
			High:      115.0,
			Low:       109.0,
			Close:     114.0,
			Volume:    1300.0,
		},
		{
			Symbol:    symbol,
			Timestamp: startTime.Add(96 * time.Hour),
			Open:      114.0,
			High:      118.0,
			Low:       112.0,
			Close:     116.0,
			Volume:    1250.0,
		},
	}
	
	// Create hourly candles
	hourlyCandles := []Candle{}
	for i := 0; i < 24*5; i++ { // 5 days of hourly data
		timestamp := startTime.Add(time.Duration(i) * time.Hour)
		basePrice := 100.0 + float64(i)*0.2
		candle := Candle{
			Symbol:    symbol,
			Timestamp: timestamp,
			Open:      basePrice,
			High:      basePrice + 1.0,
			Low:       basePrice - 0.5,
			Close:     basePrice + 0.3,
			Volume:    200.0 + float64(i%12)*10.0,
		}
		hourlyCandles = append(hourlyCandles, candle)
	}
	
	dataProvider.AddCandles(symbol, "daily", dailyCandles)
	dataProvider.AddCandles(symbol, "hourly", hourlyCandles)
	
	// Create simulation clock
	clock := NewSimulationClock(startTime, 1.0, 100*time.Millisecond)
	
	// Create simulator
	simulator := NewSimulator(clock, dataProvider)
	
	// Create WebSocket server
	wsServer := NewWebSocketServer(simulator)
	
	// Register WebSocket server as a listener for candles
	simulator.RegisterListener(wsServer)
	
	// Start HTTP server for WebSocket
	server := httptest.NewServer(http.HandlerFunc(wsServer.HandleWebSocket))
	defer server.Close()
	
	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, err := NewWebSocketClient(wsURL)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer client.Close()
	
	// Start the simulator
	startMsg := map[string]interface{}{
		"type":       "start",
		"symbol":     symbol,
		"start_time": startTime.Format(time.RFC3339),
	}
	
	err = client.SendJSON(startMsg)
	if err != nil {
		t.Fatalf("Failed to send start message: %v", err)
	}
	
	// Wait for status response
	_, err = client.ReceiveMessage(time.Second)
	if err != nil {
		t.Fatalf("Failed to receive status message: %v", err)
	}
	
	// Wait for candles with increasing timeout
	log.Printf("Waiting for candles... (this test will run for 15 seconds)")
	
	time.Sleep(15 * time.Second)
}