# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The trader_v2-simulator is a Go-based cryptocurrency trading simulator that uses historical price data to simulate daily and hourly ticks using websocket.
For websocet we are using gorilla library.
The project focuses on SOL-USD trading pairs with both daily and hourly price candles.

## Repository Structure

- `configs/`: Configuration files for the simulator
- `data/`: Contains historical price data
  - `daily/`: Daily candle data for supported symbols
  - `hourly/`: Hourly candle data for supported symbols
  - `metadata.json`: Metadata about the available price data
- `internal/`: Core application code
  - `models/`: Data models and structures
  - `simulator/`: Trading simulation logic
  - `websocket/`: WebSocket implementation for real-time data

## Development Commands

```bash
# Build the application
go build -o simulator

# Run the application
./simulator

# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/simulator/...

# Format code
go fmt ./...

# Lint code
golangci-lint run

# Update dependencies
go mod tidy
```

## Data Structure

The simulator works with cryptocurrency price data in JSON format:

- Candle data includes: timestamp, open, high, low, close, volume
- Data is organized by symbol and timeframe (daily, hourly)
- Metadata tracks symbol information, date ranges, and data counts

Example metadata structure:

```json
{
  "symbol": "SOL-USD",
  "polygon_symbol": "X:SOLUSD",
  "start_date": "2025-04-25",
  "end_date": "2025-05-18",
  "fetch_timestamp": "2025-05-16 11:00:00",
  "daily_candles_count": 24,
  "hourly_candles_count": 653
}
```
