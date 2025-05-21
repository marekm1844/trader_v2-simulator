package simulator

import (
	"fmt"
	"sync"
	"time"
)

// TimedEvent represents an event scheduled to occur at a specific simulation time
type TimedEvent struct {
	eventTime time.Time
	eventFn   func()
}

// SimulationClockImpl implements the SimulationClock interface
type SimulationClockImpl struct {
	mu              sync.RWMutex
	currentTime     time.Time
	speed           SimulationSpeed
	status          SimulationStatus
	timedEvents     []TimedEvent
	ticker          *time.Ticker
	tickerDuration  time.Duration
	tickerDoneCh    chan struct{}
	realTimeToTick  time.Duration // How much real time passes between ticks
	simulationStep  time.Duration // How much simulation time advances per tick
}

// NewSimulationClock creates a new simulation clock instance
func NewSimulationClock(initialTime time.Time, initialSpeed SimulationSpeed, tickerDuration time.Duration) *SimulationClockImpl {
	return &SimulationClockImpl{
		currentTime:    initialTime,
		speed:          initialSpeed,
		status:         StatusStopped,
		timedEvents:    make([]TimedEvent, 0),
		tickerDuration: tickerDuration,
		tickerDoneCh:   make(chan struct{}),
		simulationStep: tickerDuration,
		realTimeToTick: tickerDuration,
	}
}

// GetCurrentTime returns the current simulation time
func (c *SimulationClockImpl) GetCurrentTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentTime
}

// Advance moves the clock forward by the specified duration
func (c *SimulationClockImpl) Advance(duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusRunning {
		return fmt.Errorf("cannot manually advance clock while it is running")
	}

	c.currentTime = c.currentTime.Add(duration)
	c.processTimedEvents()
	return nil
}

// SetTime sets the clock to a specific time
func (c *SimulationClockImpl) SetTime(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusRunning {
		return fmt.Errorf("cannot set time while clock is running")
	}

	c.currentTime = t
	c.processTimedEvents()
	return nil
}

// SetSpeed adjusts how quickly simulation time advances compared to real time
func (c *SimulationClockImpl) SetSpeed(speed SimulationSpeed) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if speed <= 0 {
		return fmt.Errorf("speed must be positive")
	}

	wasRunning := c.status == StatusRunning

	// Stop current ticker if running
	if wasRunning {
		c.stopTicker()
	}

	c.speed = speed
	c.recalculateTimeStep()

	// Restart ticker if it was running
	if wasRunning {
		c.startTicker()
	}

	return nil
}

// recalculateTimeStep updates time calculations based on speed
func (c *SimulationClockImpl) recalculateTimeStep() {
	c.simulationStep = time.Duration(float64(c.tickerDuration) * float64(c.speed))
}

// RegisterTimedEvent schedules a function to be called at a specific simulation time
func (c *SimulationClockImpl) RegisterTimedEvent(eventTime time.Time, eventFn func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if eventTime.Before(c.currentTime) {
		return fmt.Errorf("cannot schedule event in the past")
	}

	c.timedEvents = append(c.timedEvents, TimedEvent{
		eventTime: eventTime,
		eventFn:   eventFn,
	})

	return nil
}

// Start begins the clock's progression
func (c *SimulationClockImpl) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusRunning {
		return fmt.Errorf("clock is already running")
	}

	c.status = StatusRunning
	c.startTicker()
	return nil
}

// startTicker creates and starts the internal ticker
func (c *SimulationClockImpl) startTicker() {
	// Safety check - Initialize the ticker channel if needed
	if c.ticker == nil {
		c.ticker = time.NewTicker(c.tickerDuration)
	}
	
	// Safety check - Initialize the done channel if needed
	if c.tickerDoneCh == nil {
		c.tickerDoneCh = make(chan struct{})
	}

	go func() {
		ticker := c.ticker // Capture the ticker to avoid a race condition
		doneCh := c.tickerDoneCh // Capture the done channel to avoid a race condition
		
		if ticker == nil || doneCh == nil {
			return // Safety guard against nil pointers
		}
		
		for {
			select {
			case <-ticker.C:
				c.tick()
			case <-doneCh:
				return
			}
		}
	}()
}

// tick advances the simulation time by one step and processes any due events
func (c *SimulationClockImpl) tick() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status != StatusRunning {
		return
	}

	c.currentTime = c.currentTime.Add(c.simulationStep)
	c.processTimedEvents()
}

// processTimedEvents checks and triggers any events that are due at the current time
func (c *SimulationClockImpl) processTimedEvents() {
	// Create a new slice to hold events that haven't been processed yet
	pendingEvents := make([]TimedEvent, 0, len(c.timedEvents))

	for _, event := range c.timedEvents {
		if !event.eventTime.After(c.currentTime) {
			// This event is due, execute it outside the lock
			go event.eventFn()
		} else {
			// This event is still in the future
			pendingEvents = append(pendingEvents, event)
		}
	}

	// Replace the events slice with only the pending events
	c.timedEvents = pendingEvents
}

// Stop halts the clock's progression
func (c *SimulationClockImpl) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusStopped {
		return fmt.Errorf("clock is already stopped")
	}

	c.status = StatusStopped
	c.stopTicker()
	return nil
}

// stopTicker stops the internal ticker
func (c *SimulationClockImpl) stopTicker() {
	// Use a local copy to avoid race conditions
	ticker := c.ticker
	doneCh := c.tickerDoneCh
	
	// Clear the fields first to prevent double-stopping
	c.ticker = nil
	c.tickerDoneCh = nil
	
	// Now stop and close using the local copies
	if ticker != nil {
		ticker.Stop()
	}
	
	if doneCh != nil {
		close(doneCh)
	}
}

// Pause temporarily halts the clock
func (c *SimulationClockImpl) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status != StatusRunning {
		return fmt.Errorf("clock is not running, current status: %s", c.status)
	}

	c.status = StatusPaused
	c.stopTicker()
	return nil
}

// Resume continues a paused clock
func (c *SimulationClockImpl) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status != StatusPaused {
		return fmt.Errorf("clock is not paused, current status: %s", c.status)
	}

	c.status = StatusRunning
	c.startTicker()
	return nil
}