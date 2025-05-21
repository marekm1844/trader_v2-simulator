package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
)

// API represents the HTTP API for the trading simulator
type API struct {
	sim simulator.Simulator
}

// NewAPI creates a new instance of the HTTP API
func NewAPI(sim simulator.Simulator) *API {
	return &API{
		sim: sim,
	}
}

// SetupRoutes configures the HTTP routes for the API
func (a *API) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/simulation", a.GetSimulationStatus).Methods("GET")
	router.HandleFunc("/api/v1/simulation", a.StartSimulation).Methods("POST")
	router.HandleFunc("/api/v1/simulation", a.StopSimulation).Methods("DELETE")
	router.HandleFunc("/api/v1/simulation/pause", a.PauseSimulation).Methods("POST")
	router.HandleFunc("/api/v1/simulation/resume", a.ResumeSimulation).Methods("POST")
	router.HandleFunc("/api/v1/simulation/speed", a.SetSimulationSpeed).Methods("PUT")
	router.HandleFunc("/api/v1/simulation/time", a.SkipToTime).Methods("PUT")

	// Add middleware for authentication, logging, and error handling
	router.Use(a.loggingMiddleware)
	// Uncomment if authentication is required
	// router.Use(a.authMiddleware)
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error"`
}

// SimulationStatusResponse represents the current simulation status
type SimulationStatusResponse struct {
	Status    string                 `json:"status"`
	Speed     float64                `json:"speed"`
	Time      time.Time              `json:"current_time"`
	Config    map[string]interface{} `json:"config,omitempty"`
	StartTime time.Time              `json:"start_time,omitempty"`
	EndTime   time.Time              `json:"end_time,omitempty"`
}

// StartSimulationRequest represents a request to start a simulation
type StartSimulationRequest struct {
	Symbol    string     `json:"symbol"`
	Timeframe string     `json:"timeframe"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Speed     *float64   `json:"speed,omitempty"`
}

// SetSpeedRequest represents a request to adjust simulation speed
type SetSpeedRequest struct {
	Speed float64 `json:"speed"`
}

// SkipToRequest represents a request to skip to a specific time
type SkipToRequest struct {
	Time time.Time `json:"time"`
}

// GetSimulationStatus returns the current state of the simulation
func (a *API) GetSimulationStatus(w http.ResponseWriter, r *http.Request) {
	status := SimulationStatusResponse{
		Status:    string(a.sim.GetStatus()),
		Speed:     float64(a.sim.GetSpeed()),
		Time:      a.sim.GetCurrentTime(),
		Config:    map[string]interface{}{}, // Populate this from simulator config
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartSimulation starts a new simulation with the provided parameters
func (a *API) StartSimulation(w http.ResponseWriter, r *http.Request) {
	var req StartSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request parameters
	if req.Symbol == "" {
		respondWithError(w, http.StatusBadRequest, "Symbol is required")
		return
	}

	if req.Timeframe != string(simulator.TimeFrameDaily) && req.Timeframe != string(simulator.TimeFrameHourly) {
		respondWithError(w, http.StatusBadRequest, "Invalid timeframe. Must be 'daily' or 'hourly'")
		return
	}

	// Set optional parameters
	if req.Speed != nil {
		if err := a.sim.SetSpeed(simulator.SimulationSpeed(*req.Speed)); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to set simulation speed: "+err.Error())
			return
		}
	}

	// Start the simulation
	if err := a.sim.Start(r.Context(), req.StartTime); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start simulation: "+err.Error())
		return
	}

	// Return the current state
	status := SimulationStatusResponse{
		Status:    string(a.sim.GetStatus()),
		Speed:     float64(a.sim.GetSpeed()),
		Time:      a.sim.GetCurrentTime(),
		StartTime: req.StartTime,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(status)
}

// StopSimulation stops the current simulation
func (a *API) StopSimulation(w http.ResponseWriter, r *http.Request) {
	if err := a.sim.Stop(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to stop simulation: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PauseSimulation pauses the current simulation
func (a *API) PauseSimulation(w http.ResponseWriter, r *http.Request) {
	if err := a.sim.Pause(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to pause simulation: "+err.Error())
		return
	}

	// Return the current state
	status := SimulationStatusResponse{
		Status: string(a.sim.GetStatus()),
		Speed:  float64(a.sim.GetSpeed()),
		Time:   a.sim.GetCurrentTime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ResumeSimulation resumes a paused simulation
func (a *API) ResumeSimulation(w http.ResponseWriter, r *http.Request) {
	if err := a.sim.Resume(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to resume simulation: "+err.Error())
		return
	}

	// Return the current state
	status := SimulationStatusResponse{
		Status: string(a.sim.GetStatus()),
		Speed:  float64(a.sim.GetSpeed()),
		Time:   a.sim.GetCurrentTime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SetSimulationSpeed changes the speed of the simulation
func (a *API) SetSimulationSpeed(w http.ResponseWriter, r *http.Request) {
	var req SetSpeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Speed <= 0 {
		respondWithError(w, http.StatusBadRequest, "Speed must be a positive number")
		return
	}

	if err := a.sim.SetSpeed(simulator.SimulationSpeed(req.Speed)); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to set simulation speed: "+err.Error())
		return
	}

	// Return the current state
	status := SimulationStatusResponse{
		Status: string(a.sim.GetStatus()),
		Speed:  float64(a.sim.GetSpeed()),
		Time:   a.sim.GetCurrentTime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SkipToTime advances the simulation to a specific point in time
func (a *API) SkipToTime(w http.ResponseWriter, r *http.Request) {
	var req SkipToRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Time.IsZero() {
		respondWithError(w, http.StatusBadRequest, "Valid target time is required")
		return
	}

	if err := a.sim.SkipTo(r.Context(), req.Time); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to skip to specified time: "+err.Error())
		return
	}

	// Return the current state
	status := SimulationStatusResponse{
		Status: string(a.sim.GetStatus()),
		Speed:  float64(a.sim.GetSpeed()),
		Time:   a.sim.GetCurrentTime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Helper function to send error responses
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// Middleware for logging requests
func (a *API) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request information
		// Log request method, path, and IP address
		next.ServeHTTP(w, r)
	})
}

// Middleware for authentication (if needed)
func (a *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add authentication logic here if required
		// For now, allow all requests
		next.ServeHTTP(w, r)
	})
}