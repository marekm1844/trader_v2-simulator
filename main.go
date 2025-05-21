package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
	"github.com/marekm1844/trader_v2-simulator/internal/websocket"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "HTTP service port")
	tickDuration := flag.Duration("tick", 100*time.Millisecond, "Simulation tick duration in milliseconds")
	flag.Parse()

	// Set up the simulator components
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	// Create the simulation clock
	clock := simulator.NewSimulationClock(
		initialTime,
		simulator.SpeedRealtime,
		*tickDuration,
	)
	
	// TODO: Replace this with a real data provider that loads from files
	mockDataProvider := createMockDataProvider([]string{"SOL-USD"})
	
	// Create simulator with default config
	config := simulator.DefaultConfig()
	sim := simulator.NewSimulator(clock, mockDataProvider, config)
	
	// Create the websocket server
	wsServer := websocket.NewServer(sim)
	wsServer.Run()
	
	// Register HTTP handlers
	http.Handle("/ws", wsServer)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Trading Simulator v2 Running"))
	})
	
	// Start the HTTP server
	log.Printf("Starting server on port %s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("HTTP server error:", err)
	}
}

// createMockDataProvider creates a temporary data provider for testing
// This would be replaced with a real implementation that loads from data files
func createMockDataProvider(symbols []string) simulator.DataProvider {
	// This is a simple mock implementation
	return &mockDataProvider{
		symbols: symbols,
	}
}

// mockDataProvider is a temporary implementation for testing
type mockDataProvider struct {
	symbols []string
}

func (m *mockDataProvider) GetCandles(ctx context.Context, symbol string, timeframe simulator.TimeFrame, 
	startTime, endTime time.Time) ([]simulator.Candle, error) {
	
	// Generate a single candle based on the requested time
	candle := simulator.Candle{
		Timestamp: startTime,
		Symbol:    symbol,
		Open:      100.0,
		High:      105.0,
		Low:       95.0,
		Close:     102.0,
		Volume:    1000.0,
	}
	
	return []simulator.Candle{candle}, nil
}

func (m *mockDataProvider) GetAvailableSymbols(ctx context.Context) ([]string, error) {
	return m.symbols, nil
}

func (m *mockDataProvider) GetDataRange(ctx context.Context, symbol string, timeframe simulator.TimeFrame) (startTime, endTime time.Time, err error) {
	// Return a fixed range for testing
	startTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	return startTime, endTime, nil
}

func (m *mockDataProvider) Reload(ctx context.Context) error {
	return nil
}