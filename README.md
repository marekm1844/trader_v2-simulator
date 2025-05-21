# Crypto Trading Simulator

A Go-based cryptocurrency trading simulator that uses historical price data to simulate daily and hourly ticks using WebSocket.

## Features

- Streams historical SOL-USD price data
- Supports both daily and hourly timeframes
- Configurable tick interval
- HTTP API for status
- Flexible date range selection

## Usage

```bash
# Build the simulator
go build -o simulator

# Run with default settings (daily candles, 1-second interval)
./simulator

# Run with hourly candles
./simulator --timeframe hourly

# Change the tick interval (e.g., 500ms)
./simulator --interval 500ms

# Specify date range
./simulator --start-date 2025-05-01 --end-date 2025-05-10

# Change HTTP server port
./simulator --port 9000

# Full example with all options
./simulator --timeframe hourly --interval 200ms --start-date 2025-05-01 --end-date 2025-05-10 --port 8888 --datadir ./data
```

## Available Parameters

| Parameter    | Default     | Description                                           |
|--------------|-------------|-------------------------------------------------------|
| --timeframe  | daily       | Timeframe to use: "daily" or "hourly"                 |
| --interval   | 1s          | Time interval between candles (e.g., 500ms, 1s, 5s)    |
| --start-date | (earliest)  | Start date for candles (format: YYYY-MM-DD)           |
| --end-date   | (latest)    | End date for candles (format: YYYY-MM-DD)             |
| --port       | 8080        | HTTP server port                                      |
| --datadir    | data        | Directory containing price data files                 |

## HTTP Endpoints

### Status Endpoint

- `GET /api/status`: Returns detailed information about the current emitter state
  - Method: GET
  - Response: JSON object with the following fields:
    - `symbol`: Trading symbol (e.g., "SOL-USD")
    - `timeframe`: Candle timeframe ("daily" or "hourly")
    - `interval`: Time between candles (e.g., "1s", "500ms")
    - `start_time`: Start date for candles (RFC3339 format)
    - `end_time`: End date for candles (RFC3339 format)
    - `is_running`: Boolean indicating if streaming is active

### Stream Control Endpoints

- `POST /api/stream/start`: Start or restart streaming with specified parameters
  - Method: POST
  - Request Body (JSON):
    ```json
    {
      "timeframe": "daily",       // Optional: "daily" or "hourly"
      "interval": "500ms",        // Optional: Duration between candles
      "start_date": "2025-05-01", // Optional: Start date (YYYY-MM-DD)
      "end_date": "2025-05-10"    // Optional: End date (YYYY-MM-DD)
    }
    ```
  - Response: JSON object with success status and message
    ```json
    {
      "success": true,
      "message": "Started streaming daily candles from 2025-05-01 to 2025-05-10 at 500ms intervals"
    }
    ```

- `POST /api/stream/stop`: Stop the current streaming session
  - Method: POST
  - Response: JSON object with success status and message
    ```json
    {
      "success": true,
      "message": "Streaming stopped successfully"
    }
    ```

## Data Structure

The simulator works with cryptocurrency price data in JSON format:

- Candle data includes: timestamp, open, high, low, close, volume
- Data is organized by symbol and timeframe (daily, hourly)
- Metadata tracks symbol information, date ranges, and data counts