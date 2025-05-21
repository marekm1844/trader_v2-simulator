package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marekm1844/trader_v2-simulator/internal/simulator"
)

// Message represents the data structure for websocket messages
type Message struct {
	Type    string      `json:"type"`
	Symbol  string      `json:"symbol,omitempty"`
	Candle  CandleData  `json:"candle,omitempty"`
	Command CommandData `json:"command,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// CandleData represents the candle data in a websocket message
type CandleData struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Symbol    string  `json:"symbol"`
	TimeFrame string  `json:"timeframe"`
}

// CommandData represents commands sent from clients
type CommandData struct {
	Action    string  `json:"action"`
	Symbol    string  `json:"symbol,omitempty"`
	StartTime string  `json:"start_time,omitempty"`
	EndTime   string  `json:"end_time,omitempty"`
	Speed     float64 `json:"speed,omitempty"`
	SkipTo    string  `json:"skip_to,omitempty"`
}

// Server manages the websocket connections and integrates with the simulator
type Server struct {
	simulator      simulator.Simulator
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	clientsMutex   sync.RWMutex
	broadcastChan  chan Message
	registerChan   chan *websocket.Conn
	unregisterChan chan *websocket.Conn
	listenerID     string
}

// NewServer creates a new websocket server
func NewServer(sim simulator.Simulator) *Server {
	server := &Server{
		simulator: sim,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections
			},
		},
		clients:        make(map[*websocket.Conn]bool),
		broadcastChan:  make(chan Message, 256),
		registerChan:   make(chan *websocket.Conn),
		unregisterChan: make(chan *websocket.Conn),
		listenerID:     "websocket-server",
	}

	// Register as a listener to the simulator
	err := sim.RegisterListener(server)
	if err != nil {
		log.Printf("Failed to register websocket server as listener: %v", err)
	}

	return server
}

// GetListenerID implements simulator.CandleListener
func (s *Server) GetListenerID() string {
	return s.listenerID
}

// OnCandle implements simulator.CandleListener
func (s *Server) OnCandle(candle simulator.Candle) {
	message := Message{
		Type:   "candle",
		Symbol: candle.Symbol,
		Candle: CandleData{
			Timestamp: candle.Timestamp.Unix(),
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			Symbol:    candle.Symbol,
			TimeFrame: string(determineTimeFrame(candle)),
		},
	}

	s.broadcastChan <- message
}

// determineTimeFrame guesses the timeframe based on the candle timestamp
func determineTimeFrame(candle simulator.Candle) simulator.TimeFrame {
	// If timestamp is at exactly midnight, assume daily
	if candle.Timestamp.Hour() == 0 && candle.Timestamp.Minute() == 0 {
		return simulator.TimeFrameDaily
	}
	return simulator.TimeFrameHourly
}

// HandleConnection manages a websocket client connection
func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}

	// Register the new client
	s.registerChan <- conn

	// Inform the client about the current state
	statusMsg := Message{
		Type: "status",
		Command: CommandData{
			Action: string(s.simulator.GetStatus()),
			Speed:  float64(s.simulator.GetSpeed()),
		},
	}

	err = conn.WriteJSON(statusMsg)
	if err != nil {
		log.Println("Error sending status:", err)
		s.unregisterChan <- conn
		return
	}

	// Handle incoming messages in a separate goroutine
	go s.readPump(conn)
}

// readPump handles messages from the client
func (s *Server) readPump(conn *websocket.Conn) {
	defer func() {
		s.unregisterChan <- conn
		conn.Close()
	}()

	conn.SetReadLimit(512) // Set max message size

	// Configure read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(msgBytes, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Process commands
		if message.Type == "command" {
			s.handleCommand(conn, message.Command)
		}
	}
}

// handleCommand processes commands from clients
func (s *Server) handleCommand(conn *websocket.Conn, cmd CommandData) {
	ctx := context.Background()
	var err error

	switch cmd.Action {
	case "start":
		startTime := time.Now()
		if cmd.StartTime != "" {
			startTime, err = time.Parse(time.RFC3339, cmd.StartTime)
			if err != nil {
				s.sendError(conn, fmt.Sprintf("Invalid start time: %v", err))
				return
			}
		}
		err = s.simulator.Start(ctx, startTime)

	case "stop":
		err = s.simulator.Stop(ctx)

	case "pause":
		err = s.simulator.Pause(ctx)

	case "resume":
		err = s.simulator.Resume(ctx)

	case "speed":
		err = s.simulator.SetSpeed(simulator.SimulationSpeed(cmd.Speed))

	case "skip_to":
		targetTime, err := time.Parse(time.RFC3339, cmd.SkipTo)
		if err != nil {
			s.sendError(conn, fmt.Sprintf("Invalid skip time: %v", err))
			return
		}
		err = s.simulator.SkipTo(ctx, targetTime)

	default:
		s.sendError(conn, fmt.Sprintf("Unknown command: %s", cmd.Action))
		return
	}

	if err != nil {
		s.sendError(conn, fmt.Sprintf("Error executing command: %v", err))
		return
	}

	// Update all clients with the new status
	s.broadcastStatus()
}

// sendError sends an error message to a specific client
func (s *Server) sendError(conn *websocket.Conn, errorMsg string) {
	errorMessage := Message{
		Type:  "error",
		Error: errorMsg,
	}

	err := conn.WriteJSON(errorMessage)
	if err != nil {
		log.Printf("Error sending error message: %v", err)
	}
}

// broadcastStatus sends the current simulator status to all clients
func (s *Server) broadcastStatus() {
	message := Message{
		Type: "status",
		Command: CommandData{
			Action: string(s.simulator.GetStatus()),
			Speed:  float64(s.simulator.GetSpeed()),
		},
	}

	s.broadcastChan <- message
}

// Run starts the server's message handling goroutines
func (s *Server) Run() {
	go s.handleChannels()
}

// handleChannels manages the channel operations in a single goroutine
func (s *Server) handleChannels() {
	for {
		select {
		case client := <-s.registerChan:
			s.clientsMutex.Lock()
			s.clients[client] = true
			s.clientsMutex.Unlock()

		case client := <-s.unregisterChan:
			s.clientsMutex.Lock()
			delete(s.clients, client)
			s.clientsMutex.Unlock()

		case message := <-s.broadcastChan:
			s.clientsMutex.RLock()
			for client := range s.clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("Error broadcasting to client: %v", err)
					client.Close()
					s.unregisterChan <- client
				}
			}
			s.clientsMutex.RUnlock()
		}
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.HandleConnection(w, r)
}