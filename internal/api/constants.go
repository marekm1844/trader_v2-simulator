package api

import (
	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
)

// SimulationConfig represents the configuration for a simulation
type SimulationConfig struct {
	Symbol    string                  `json:"symbol"`
	Timeframe simulator.TimeFrame     `json:"timeframe"`
	Speed     simulator.SimulationSpeed `json:"speed"`
}

// These constants represent the standard timeframes supported by the API
const (
	// API Endpoint paths
	APIBasePath         = "/api/v1"
	SimulationEndpoint  = APIBasePath + "/simulation"
	PauseEndpoint       = SimulationEndpoint + "/pause"
	ResumeEndpoint      = SimulationEndpoint + "/resume"
	SpeedEndpoint       = SimulationEndpoint + "/speed"
	TimeEndpoint        = SimulationEndpoint + "/time"
	
	// Documentation constants
	APIVersion          = "1.0.0"
	
	// Minimum and maximum values for simulation parameters
	MinSimulationSpeed  = 0.1
	MaxSimulationSpeed  = 100.0
)

/*
# Trading Simulator REST API

## Overview

The Trading Simulator API allows clients to control the simulation of trading data through a REST interface.
All endpoints use JSON for request and response payloads.

## Authentication

Currently, the API does not require authentication. This may change in future versions.

## Rate Limiting

A basic rate limit of 100 requests per minute per IP address is enforced.

## Endpoints

### GET /api/v1/simulation

Returns the current status of the simulation.

**Response:**
```json
{
  "status": "running|paused|stopped",
  "speed": 1.0,
  "current_time": "2025-04-15T14:30:00Z",
  "config": {
    "symbol": "SOL-USD",
    "timeframe": "daily|hourly"
  },
  "start_time": "2025-01-01T00:00:00Z",
  "end_time": "2025-12-31T23:59:59Z"
}
```

### POST /api/v1/simulation

Starts a new simulation with the specified parameters.

**Request:**
```json
{
  "symbol": "SOL-USD",
  "timeframe": "daily|hourly",
  "start_time": "2025-01-01T00:00:00Z",
  "end_time": "2025-12-31T23:59:59Z",  // Optional
  "speed": 1.0                         // Optional
}
```

**Response:**
```json
{
  "status": "running",
  "speed": 1.0,
  "current_time": "2025-01-01T00:00:00Z",
  "start_time": "2025-01-01T00:00:00Z"
}
```

### DELETE /api/v1/simulation

Stops the current simulation.

**Response:** 204 No Content

### POST /api/v1/simulation/pause

Pauses the running simulation.

**Response:**
```json
{
  "status": "paused",
  "speed": 1.0,
  "current_time": "2025-02-15T10:30:00Z"
}
```

### POST /api/v1/simulation/resume

Resumes a paused simulation.

**Response:**
```json
{
  "status": "running",
  "speed": 1.0,
  "current_time": "2025-02-15T10:30:00Z"
}
```

### PUT /api/v1/simulation/speed

Adjusts the simulation speed.

**Request:**
```json
{
  "speed": 5.0  // Speed multiplier (0.1 to 100.0)
}
```

**Response:**
```json
{
  "status": "running",
  "speed": 5.0,
  "current_time": "2025-02-15T10:30:00Z"
}
```

### PUT /api/v1/simulation/time

Jumps to a specific point in time within the simulation.

**Request:**
```json
{
  "time": "2025-03-01T00:00:00Z"
}
```

**Response:**
```json
{
  "status": "running",
  "speed": 1.0,
  "current_time": "2025-03-01T00:00:00Z"
}
```

## WebSocket API

In addition to the REST API, clients can connect to the `/ws` endpoint to receive real-time candle data and send commands.

### Connection

Connect to `ws://<host>:<port>/ws`

### Messages

**Candle Updates:**
```json
{
  "type": "candle",
  "symbol": "SOL-USD",
  "candle": {
    "timestamp": 1672531200,
    "open": 100.50,
    "high": 105.75,
    "low": 99.25,
    "close": 102.30,
    "volume": 1050000,
    "symbol": "SOL-USD",
    "timeframe": "daily|hourly"
  }
}
```

**Status Updates:**
```json
{
  "type": "status",
  "command": {
    "action": "running|paused|stopped",
    "speed": 1.0
  }
}
```

**Commands (sent from client to server):**
```json
{
  "type": "command",
  "command": {
    "action": "start|stop|pause|resume|speed|skip_to",
    "symbol": "SOL-USD",           // For start
    "start_time": "2025-01-01T00:00:00Z",  // For start
    "speed": 2.0,                  // For speed
    "skip_to": "2025-03-01T00:00:00Z"      // For skip_to
  }
}
```

**Error Messages:**
```json
{
  "type": "error",
  "error": "Error message describing the issue"
}
```
*/