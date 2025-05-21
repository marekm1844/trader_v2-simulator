package simulator

import (
	"sync"
	"testing"
	"time"
)

func TestSimulationClockBasicFunctionality(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tickerDuration := 10 * time.Millisecond
	clock := NewSimulationClock(initialTime, SpeedRealtime, tickerDuration)

	// Test initial state
	if !clock.GetCurrentTime().Equal(initialTime) {
		t.Errorf("Expected initial time %v, got %v", initialTime, clock.GetCurrentTime())
	}

	// Test manual time advance
	advanceDuration := 1 * time.Hour
	err := clock.Advance(advanceDuration)
	if err != nil {
		t.Errorf("Advance failed: %v", err)
	}

	expectedTime := initialTime.Add(advanceDuration)
	if !clock.GetCurrentTime().Equal(expectedTime) {
		t.Errorf("Expected time after advance %v, got %v", expectedTime, clock.GetCurrentTime())
	}

	// Test SetTime
	newTime := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	err = clock.SetTime(newTime)
	if err != nil {
		t.Errorf("SetTime failed: %v", err)
	}

	if !clock.GetCurrentTime().Equal(newTime) {
		t.Errorf("Expected time after SetTime %v, got %v", newTime, clock.GetCurrentTime())
	}
}

func TestSimulationClockRunning(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tickerDuration := 10 * time.Millisecond
	clock := NewSimulationClock(initialTime, SpeedFast, tickerDuration)

	// Start the clock
	err := clock.Start()
	if err != nil {
		t.Fatalf("Failed to start clock: %v", err)
	}

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// Current time should have advanced
	currentTime := clock.GetCurrentTime()
	if !currentTime.After(initialTime) {
		t.Errorf("Clock did not advance: initial=%v, current=%v", initialTime, currentTime)
	}

	// Test that we can't manually advance while running
	err = clock.Advance(1 * time.Hour)
	if err == nil {
		t.Error("Expected error when advancing running clock, got nil")
	}

	// Test that we can't manually set time while running
	err = clock.SetTime(initialTime)
	if err == nil {
		t.Error("Expected error when setting time on running clock, got nil")
	}

	// Stop the clock
	err = clock.Stop()
	if err != nil {
		t.Fatalf("Failed to stop clock: %v", err)
	}
}

func TestSimulationClockSpeedChanges(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tickerDuration := 10 * time.Millisecond
	clock := NewSimulationClock(initialTime, SpeedRealtime, tickerDuration)

	// Change speed before starting
	err := clock.SetSpeed(SpeedFast)
	if err != nil {
		t.Fatalf("Failed to set speed: %v", err)
	}

	// Start the clock
	err = clock.Start()
	if err != nil {
		t.Fatalf("Failed to start clock: %v", err)
	}

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)
	timeAfterFast := clock.GetCurrentTime()

	// Change to slow speed
	err = clock.SetSpeed(SpeedSlow)
	if err != nil {
		t.Fatalf("Failed to change speed: %v", err)
	}

	// Let it run some more
	time.Sleep(50 * time.Millisecond)
	timeAfterSlow := clock.GetCurrentTime()

	// Calculate advancement rates
	fastAdvancement := timeAfterFast.Sub(initialTime)
	slowAdvancement := timeAfterSlow.Sub(timeAfterFast)

	// The fast advancement should be greater than slow for the same real time period
	// But we need to account for timing variations in the test
	if fastAdvancement == 0 || slowAdvancement == 0 {
		t.Fatalf("Clock didn't advance: fast=%v, slow=%v", fastAdvancement, slowAdvancement)
	}

	fastRate := float64(fastAdvancement) / 50.0
	slowRate := float64(slowAdvancement) / 50.0

	if slowRate >= fastRate {
		t.Errorf("Expected slow rate to be less than fast rate: fast=%v/ms, slow=%v/ms", 
			fastRate/float64(time.Millisecond), slowRate/float64(time.Millisecond))
	}

	// Stop the clock
	err = clock.Stop()
	if err != nil {
		t.Fatalf("Failed to stop clock: %v", err)
	}
}

func TestSimulationClockTimedEvents(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tickerDuration := 10 * time.Millisecond
	clock := NewSimulationClock(initialTime, SpeedRealtime, tickerDuration)

	// Event times
	eventTime1 := initialTime.Add(30 * time.Millisecond)
	eventTime2 := initialTime.Add(60 * time.Millisecond)
	
	// Track events being called
	var wg sync.WaitGroup
	wg.Add(2)
	
	event1Called := false
	event2Called := false
	
	// Register events
	_ = clock.RegisterTimedEvent(eventTime1, func() {
		event1Called = true
		wg.Done()
	})
	
	_ = clock.RegisterTimedEvent(eventTime2, func() {
		event2Called = true
		wg.Done()
	})
	
	// Start the clock
	_ = clock.Start()
	
	// Wait for events to be called with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Events were called
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for events")
	}
	
	// Stop the clock
	_ = clock.Stop()
	
	// Check that both events were called
	if !event1Called {
		t.Error("Event 1 was not called")
	}
	
	if !event2Called {
		t.Error("Event 2 was not called")
	}
}

func TestSimulationClockPauseResume(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tickerDuration := 10 * time.Millisecond
	clock := NewSimulationClock(initialTime, SpeedRealtime, tickerDuration)

	// Start the clock
	err := clock.Start()
	if err != nil {
		t.Fatalf("Failed to start clock: %v", err)
	}

	// Let it run for a bit
	time.Sleep(30 * time.Millisecond)
	timeBeforePause := clock.GetCurrentTime()
	
	// Pause the clock
	err = clock.Pause()
	if err != nil {
		t.Fatalf("Failed to pause clock: %v", err)
	}
	
	// Wait a bit while paused
	time.Sleep(50 * time.Millisecond)
	timeAfterPause := clock.GetCurrentTime()
	
	// Time should not have changed while paused
	if !timeBeforePause.Equal(timeAfterPause) {
		t.Errorf("Time advanced while paused: before=%v, after=%v", 
			timeBeforePause, timeAfterPause)
	}
	
	// Resume the clock
	err = clock.Resume()
	if err != nil {
		t.Fatalf("Failed to resume clock: %v", err)
	}
	
	// Let it run some more
	time.Sleep(30 * time.Millisecond)
	timeAfterResume := clock.GetCurrentTime()
	
	// Time should have advanced after resuming
	if !timeAfterResume.After(timeAfterPause) {
		t.Errorf("Time did not advance after resume: pause=%v, resume=%v", 
			timeAfterPause, timeAfterResume)
	}
	
	// Stop the clock
	err = clock.Stop()
	if err != nil {
		t.Fatalf("Failed to stop clock: %v", err)
	}
}