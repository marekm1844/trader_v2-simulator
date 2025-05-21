package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
	"github.com/marekm1844/trader_v2-simulator/internal/simulator/dataprovider"
)

// ConsoleEmitter emits candles to the console at the specified interval
type ConsoleEmitter struct {
	timeframe    simulator.TimeFrame
	symbol       string
	interval     time.Duration
	dataProvider simulator.DataProvider
	endTime      time.Time      // End time to stop at
	stopCh       chan struct{}
}

// NewConsoleEmitter creates a new console candle emitter
func NewConsoleEmitter(
	symbol string, 
	timeframe simulator.TimeFrame, 
	interval time.Duration, 
	dataProvider simulator.DataProvider,
	endTime time.Time,
) *ConsoleEmitter {
	return &ConsoleEmitter{
		timeframe:    timeframe,
		symbol:       symbol,
		interval:     interval,
		dataProvider: dataProvider,
		endTime:      endTime,
		stopCh:       make(chan struct{}),
	}
}

// Start begins emitting candles at the specified interval
func (c *ConsoleEmitter) Start(ctx context.Context, startTime time.Time) {
	log.Printf("Starting candle emitter for %s [%s] at %v intervals", 
		c.symbol, c.timeframe, c.interval)
	
	// Get the data range
	dataStart, dataEnd, err := c.dataProvider.GetDataRange(ctx, c.symbol, c.timeframe)
	if err != nil {
		log.Printf("Error getting data range: %v", err)
		return
	}
	
	log.Printf("Data available from %s to %s", 
		dataStart.Format("2006-01-02"), dataEnd.Format("2006-01-02"))
	
	// Use the later of dataStart and startTime
	currentTime := startTime
	if dataStart.After(startTime) {
		currentTime = dataStart
	}
	
	// Start ticker to emit candles
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check if we've reached the user-specified end time
			if currentTime.After(c.endTime) {
				log.Printf("Reached specified end time: %s, stopping emitter", 
					c.endTime.Format("2006-01-02 15:04:05"))
				return
			}
			
			// Get the next candle (check if we've reached the end of available data)
			if currentTime.After(dataEnd) {
				log.Println("Reached end of data, restarting from beginning")
				currentTime = dataStart
			}
			
			// Set window size based on timeframe
			var windowSize time.Duration
			var timeAdvance time.Duration
			
			if c.timeframe == simulator.TimeFrameHourly {
				windowSize = 2 * time.Hour   // 2-hour window for hourly candles
				timeAdvance = 1 * time.Hour  // Advance by 1 hour for hourly candles
			} else {
				windowSize = 48 * time.Hour  // 48-hour window for daily candles
				timeAdvance = 24 * time.Hour // Advance by 24 hours for daily candles
			}
			
			endWindow := currentTime.Add(windowSize)
			candles, err := c.dataProvider.GetCandles(ctx, c.symbol, c.timeframe, 
				currentTime, endWindow)
			if err != nil {
				log.Printf("Error getting candles: %v", err)
				continue
			}
			
			if len(candles) == 0 {
				log.Printf("No candles available at %s", 
					currentTime.Format("2006-01-02 15:04:05"))
				currentTime = currentTime.Add(timeAdvance)
				continue
			}
			
			// Emit the first candle
			candle := candles[0]
			log.Printf("CANDLE [%s] %s: O: %.2f, H: %.2f, L: %.2f, C: %.2f, V: %.2f",
				c.timeframe, candle.Timestamp.Format(time.RFC3339),
				candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
			
			// Move to the next time period
			currentTime = currentTime.Add(timeAdvance)
			
		case <-c.stopCh:
			log.Println("Stopping candle emitter")
			return
		case <-ctx.Done():
			log.Println("Context canceled, stopping candle emitter")
			return
		}
	}
}

// Stop halts candle emission
func (c *ConsoleEmitter) Stop() {
	close(c.stopCh)
}

func main() {
	// Parse command line flags
	timeframeFlag := flag.String("timeframe", "daily", "Timeframe to use (daily or hourly)")
	tickInterval := flag.Duration("interval", 1*time.Second, "Interval between candles (default 1s)")
	dataDir := flag.String("datadir", "data", "Directory containing price data files")
	port := flag.Int("port", 8080, "HTTP server port")
	startDateStr := flag.String("start-date", "", "Start date for candles (format: 2025-01-02, defaults to earliest available)")
	endDateStr := flag.String("end-date", "", "End date for candles (format: 2025-01-02, defaults to latest available)")
	flag.Parse()

	// Validate and parse timeframe
	var timeframe simulator.TimeFrame
	switch *timeframeFlag {
	case "daily":
		timeframe = simulator.TimeFrameDaily
	case "hourly":
		timeframe = simulator.TimeFrameHourly
	default:
		log.Fatalf("Invalid timeframe: %s. Must be 'daily' or 'hourly'", *timeframeFlag)
	}

	log.Printf("Starting simulator with %s candles every %v", timeframe, *tickInterval)

	// Get absolute path for data directory
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for data directory: %v", err)
	}
	
	// Create file data provider
	dataProvider, err := dataprovider.NewFileDataProvider(absDataDir)
	if err != nil {
		log.Fatalf("Failed to create file data provider: %v", err)
	}
	
	// Get available symbols
	symbols, err := dataProvider.GetAvailableSymbols(context.Background())
	if err != nil {
		log.Fatalf("Failed to get available symbols: %v", err)
	}
	
	if len(symbols) == 0 {
		log.Fatalf("No trading symbols available in data directory")
	}
	
	symbol := symbols[0]
	log.Printf("Using symbol: %s", symbol)
	
	// Get data range to determine available dates
	dataStart, dataEnd, err := dataProvider.GetDataRange(context.Background(), symbol, timeframe)
	if err != nil {
		log.Fatalf("Failed to get data range: %v", err)
	}
	
	// Parse user-specified start date if provided
	var initialTime time.Time
	if *startDateStr != "" {
		parsedStartDate, err := time.Parse("2006-01-02", *startDateStr)
		if err != nil {
			log.Fatalf("Invalid start-date format. Use YYYY-MM-DD format: %v", err)
		}
		initialTime = parsedStartDate
		
		// Ensure start date is within available data range
		if initialTime.Before(dataStart) {
			log.Printf("Warning: Specified start-date %s is before earliest available data (%s). Using earliest available date.",
				initialTime.Format("2006-01-02"), dataStart.Format("2006-01-02"))
			initialTime = dataStart
		}
		if initialTime.After(dataEnd) {
			log.Fatalf("Error: Specified start-date %s is after latest available data (%s)",
				initialTime.Format("2006-01-02"), dataEnd.Format("2006-01-02"))
		}
	} else {
		// Use the data start time as initial time if not specified
		initialTime = dataStart
	}
	
	// Parse user-specified end date if provided
	var endTime time.Time
	if *endDateStr != "" {
		parsedEndDate, err := time.Parse("2006-01-02", *endDateStr)
		if err != nil {
			log.Fatalf("Invalid end-date format. Use YYYY-MM-DD format: %v", err)
		}
		endTime = parsedEndDate
		
		// Ensure end date is within available data range
		if endTime.After(dataEnd) {
			log.Printf("Warning: Specified end-date %s is after latest available data (%s). Using latest available date.",
				endTime.Format("2006-01-02"), dataEnd.Format("2006-01-02"))
			endTime = dataEnd
		}
		if endTime.Before(dataStart) {
			log.Fatalf("Error: Specified end-date %s is before earliest available data (%s)",
				endTime.Format("2006-01-02"), dataStart.Format("2006-01-02"))
		}
		if endTime.Before(initialTime) {
			log.Fatalf("Error: end-date (%s) must be after start-date (%s)",
				endTime.Format("2006-01-02"), initialTime.Format("2006-01-02"))
		}
	} else {
		// Use the data end time as end time if not specified
		endTime = dataEnd
	}
	
	log.Printf("Using date range: %s to %s", initialTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	
	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create a console emitter for the selected timeframe
	emitter := NewConsoleEmitter(
		symbol,
		timeframe, 
		*tickInterval,
		dataProvider,
		endTime,
	)
	
	// Start the emitter in a goroutine
	go emitter.Start(ctx, initialTime)
	
	log.Println("Simulator started. Press Ctrl+C to stop.")
	
	// Create a status endpoint for checking if the server is running
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		statusMsg := fmt.Sprintf("Trading Simulator Running - Streaming %s candles for %s from %s to %s", 
			timeframe, symbol, initialTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
		w.Write([]byte(statusMsg))
	})
	
	// Start the HTTP server in a goroutine
	go func() {
		serverAddr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting HTTP server on %s", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil {
			log.Println("HTTP server error:", err)
		}
	}()
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	// Stop all components
	log.Println("Stopping simulator...")
	
	// Cancel the context to signal all goroutines to stop
	cancel()
	
	// Explicitly stop the emitter
	emitter.Stop()
	
	log.Println("Simulator stopped")
}