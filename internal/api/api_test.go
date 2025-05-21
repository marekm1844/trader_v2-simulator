package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
	"github.com/stretchr/testify/assert"
)

// MockSimulator is a simple stub implementation of the Simulator interface for testing
type MockSimulator struct {
	status        simulator.SimulationStatus
	speed         simulator.SimulationSpeed
	currentTime   time.Time
	startError    error
	stopError     error
	pauseError    error
	resumeError   error
	speedError    error
	skipError     error
	registerError error
}

func NewMockSimulator() *MockSimulator {
	return &MockSimulator{
		status:      simulator.StatusStopped,
		speed:       simulator.SpeedRealtime,
		currentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (m *MockSimulator) SetMockErrors(startErr, stopErr, pauseErr, resumeErr, speedErr, skipErr, regErr error) {
	m.startError = startErr
	m.stopError = stopErr
	m.pauseError = pauseErr
	m.resumeError = resumeErr
	m.speedError = speedErr
	m.skipError = skipErr
	m.registerError = regErr
}

func (m *MockSimulator) Start(ctx context.Context, startTime time.Time) error {
	if m.startError != nil {
		return m.startError
	}
	m.status = simulator.StatusRunning
	m.currentTime = startTime
	return nil
}

func (m *MockSimulator) Stop(ctx context.Context) error {
	if m.stopError != nil {
		return m.stopError
	}
	m.status = simulator.StatusStopped
	return nil
}

func (m *MockSimulator) Pause(ctx context.Context) error {
	if m.pauseError != nil {
		return m.pauseError
	}
	m.status = simulator.StatusPaused
	return nil
}

func (m *MockSimulator) Resume(ctx context.Context) error {
	if m.resumeError != nil {
		return m.resumeError
	}
	m.status = simulator.StatusRunning
	return nil
}

func (m *MockSimulator) SetSpeed(speed simulator.SimulationSpeed) error {
	if m.speedError != nil {
		return m.speedError
	}
	m.speed = speed
	return nil
}

func (m *MockSimulator) GetSpeed() simulator.SimulationSpeed {
	return m.speed
}

func (m *MockSimulator) GetStatus() simulator.SimulationStatus {
	return m.status
}

func (m *MockSimulator) GetCurrentTime() time.Time {
	return m.currentTime
}

func (m *MockSimulator) RegisterListener(listener simulator.CandleListener) error {
	if m.registerError != nil {
		return m.registerError
	}
	return nil
}

func (m *MockSimulator) UnregisterListener(listener simulator.CandleListener) error {
	return nil
}

func (m *MockSimulator) SkipTo(ctx context.Context, targetTime time.Time) error {
	if m.skipError != nil {
		return m.skipError
	}
	m.currentTime = targetTime
	return nil
}

// TestGetSimulationStatus tests the GetSimulationStatus handler
func TestGetSimulationStatus(t *testing.T) {
	// Create a mock simulator
	mockSim := NewMockSimulator()
	mockSim.status = simulator.StatusRunning
	
	// Create the API handler
	api := NewAPI(mockSim)
	
	// Create a request
	req, err := http.NewRequest("GET", "/api/v1/simulation", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create a response recorder
	rr := httptest.NewRecorder()
	
	// Call the handler
	http.HandlerFunc(api.GetSimulationStatus).ServeHTTP(rr, req)
	
	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)
	
	// Parse the response
	var resp SimulationStatusResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	
	// Check the response
	assert.Equal(t, string(simulator.StatusRunning), resp.Status)
	assert.Equal(t, float64(simulator.SpeedRealtime), resp.Speed)
}

// Create request struct specific for this test to match the API shape
// without redeclaring the type from api.go

// TestStartSimulation tests the StartSimulation handler
func TestStartSimulation(t *testing.T) {
	// Create a mock simulator
	mockSim := NewMockSimulator()
	
	// Create the API handler
	api := NewAPI(mockSim)
	
	// Create a request body
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	speed := 1.0
	reqBody := struct {
		Symbol    string    `json:"symbol"`
		Timeframe string    `json:"timeframe"`
		StartTime time.Time `json:"start_time"`
		Speed     *float64  `json:"speed,omitempty"`
	}{
		Symbol:    "SOL-USD",
		Timeframe: "daily",
		StartTime: startTime,
		Speed:     &speed,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	
	// Create a request
	req, err := http.NewRequest("POST", "/api/v1/simulation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Create a response recorder
	rr := httptest.NewRecorder()
	
	// Set up a minimal router to handle the request
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/simulation", api.StartSimulation).Methods("POST")
	router.ServeHTTP(rr, req)
	
	// Check the status code
	assert.Equal(t, http.StatusCreated, rr.Code)
	
	// Parse the response
	var resp SimulationStatusResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	
	// Check the response
	assert.Equal(t, string(simulator.StatusRunning), resp.Status)
	assert.Equal(t, float64(simulator.SpeedRealtime), resp.Speed)
}

// TestStartSimulationError tests error handling in the StartSimulation handler
func TestStartSimulationError(t *testing.T) {
	// Create a mock simulator with an error
	mockSim := NewMockSimulator()
	mockSim.SetMockErrors(fmt.Errorf("start error"), nil, nil, nil, nil, nil, nil)
	
	// Create the API handler
	api := NewAPI(mockSim)
	
	// Create a request body
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	speed := 1.0
	reqBody := struct {
		Symbol    string    `json:"symbol"`
		Timeframe string    `json:"timeframe"`
		StartTime time.Time `json:"start_time"`
		Speed     *float64  `json:"speed,omitempty"`
	}{
		Symbol:    "SOL-USD",
		Timeframe: "daily",
		StartTime: startTime,
		Speed:     &speed,
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	
	// Create a request
	req, err := http.NewRequest("POST", "/api/v1/simulation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Create a response recorder
	rr := httptest.NewRecorder()
	
	// Call the handler
	http.HandlerFunc(api.StartSimulation).ServeHTTP(rr, req)
	
	// Check the status code is an error
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}