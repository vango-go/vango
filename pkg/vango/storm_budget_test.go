package vango

import (
	"testing"
	"time"
)

func TestStormBudgetTrackerNil(t *testing.T) {
	var tracker *StormBudgetTracker

	// All methods should be safe to call on nil
	if err := tracker.CheckResource(); err != nil {
		t.Errorf("CheckResource on nil should return nil, got %v", err)
	}
	if err := tracker.CheckAction(); err != nil {
		t.Errorf("CheckAction on nil should return nil, got %v", err)
	}
	if err := tracker.CheckGoLatest(); err != nil {
		t.Errorf("CheckGoLatest on nil should return nil, got %v", err)
	}
	if err := tracker.CheckEffectRun(); err != nil {
		t.Errorf("CheckEffectRun on nil should return nil, got %v", err)
	}

	// Should not panic
	tracker.ResetTick()
}

func TestStormBudgetResourceLimit(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxResourceStartsPerSecond: 3,
		WindowDuration:             100 * time.Millisecond,
	})

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		if err := tracker.CheckResource(); err != nil {
			t.Errorf("CheckResource %d should succeed, got %v", i+1, err)
		}
	}

	// 4th should fail
	if err := tracker.CheckResource(); err != ErrBudgetExceeded {
		t.Errorf("CheckResource 4 should return ErrBudgetExceeded, got %v", err)
	}

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)

	// Should succeed again
	if err := tracker.CheckResource(); err != nil {
		t.Errorf("CheckResource after window should succeed, got %v", err)
	}
}

func TestStormBudgetActionLimit(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxActionStartsPerSecond: 2,
		WindowDuration:           100 * time.Millisecond,
	})

	// First 2 should succeed
	for i := 0; i < 2; i++ {
		if err := tracker.CheckAction(); err != nil {
			t.Errorf("CheckAction %d should succeed, got %v", i+1, err)
		}
	}

	// 3rd should fail
	if err := tracker.CheckAction(); err != ErrBudgetExceeded {
		t.Errorf("CheckAction 3 should return ErrBudgetExceeded, got %v", err)
	}
}

func TestStormBudgetGoLatestLimit(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxGoLatestStartsPerSecond: 2,
		WindowDuration:             100 * time.Millisecond,
	})

	// First 2 should succeed
	for i := 0; i < 2; i++ {
		if err := tracker.CheckGoLatest(); err != nil {
			t.Errorf("CheckGoLatest %d should succeed, got %v", i+1, err)
		}
	}

	// 3rd should fail
	if err := tracker.CheckGoLatest(); err != ErrBudgetExceeded {
		t.Errorf("CheckGoLatest 3 should return ErrBudgetExceeded, got %v", err)
	}
}

func TestStormBudgetEffectRunsPerTick(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxEffectRunsPerTick: 3,
	})

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		if err := tracker.CheckEffectRun(); err != nil {
			t.Errorf("CheckEffectRun %d should succeed, got %v", i+1, err)
		}
	}

	// 4th should fail
	if err := tracker.CheckEffectRun(); err != ErrBudgetExceeded {
		t.Errorf("CheckEffectRun 4 should return ErrBudgetExceeded, got %v", err)
	}

	// Reset tick
	tracker.ResetTick()

	// Should succeed again
	if err := tracker.CheckEffectRun(); err != nil {
		t.Errorf("CheckEffectRun after ResetTick should succeed, got %v", err)
	}
}

func TestStormBudgetUnlimited(t *testing.T) {
	// 0 means unlimited
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxResourceStartsPerSecond: 0,
		MaxActionStartsPerSecond:   0,
		MaxGoLatestStartsPerSecond: 0,
		MaxEffectRunsPerTick:       0,
	})

	// All should succeed unlimited times
	for i := 0; i < 1000; i++ {
		if err := tracker.CheckResource(); err != nil {
			t.Errorf("CheckResource should always succeed with 0 limit, got %v", err)
		}
		if err := tracker.CheckAction(); err != nil {
			t.Errorf("CheckAction should always succeed with 0 limit, got %v", err)
		}
		if err := tracker.CheckGoLatest(); err != nil {
			t.Errorf("CheckGoLatest should always succeed with 0 limit, got %v", err)
		}
		if err := tracker.CheckEffectRun(); err != nil {
			t.Errorf("CheckEffectRun should always succeed with 0 limit, got %v", err)
		}
	}
}

func TestStormBudgetStats(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxResourceStartsPerSecond: 10,
		MaxActionStartsPerSecond:   10,
		MaxGoLatestStartsPerSecond: 10,
		MaxEffectRunsPerTick:       10,
		WindowDuration:             time.Second,
	})

	// Make some calls
	tracker.CheckResource()
	tracker.CheckResource()
	tracker.CheckAction()
	tracker.CheckGoLatest()
	tracker.CheckGoLatest()
	tracker.CheckGoLatest()
	tracker.CheckEffectRun()

	stats := tracker.Stats()

	if stats.ResourceStartsInWindow != 2 {
		t.Errorf("ResourceStartsInWindow = %d, want 2", stats.ResourceStartsInWindow)
	}
	if stats.ActionStartsInWindow != 1 {
		t.Errorf("ActionStartsInWindow = %d, want 1", stats.ActionStartsInWindow)
	}
	if stats.GoLatestStartsInWindow != 3 {
		t.Errorf("GoLatestStartsInWindow = %d, want 3", stats.GoLatestStartsInWindow)
	}
	if stats.EffectRunsThisTick != 1 {
		t.Errorf("EffectRunsThisTick = %d, want 1", stats.EffectRunsThisTick)
	}
}

func TestStormBudgetGetOnExceeded(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		OnExceeded: BudgetModeTripBreaker,
	})

	if tracker.GetOnExceeded() != BudgetModeTripBreaker {
		t.Errorf("GetOnExceeded = %v, want BudgetModeTripBreaker", tracker.GetOnExceeded())
	}

	var nilTracker *StormBudgetTracker
	if nilTracker.GetOnExceeded() != BudgetModeThrottle {
		t.Errorf("nil tracker GetOnExceeded = %v, want BudgetModeThrottle", nilTracker.GetOnExceeded())
	}
}

func TestSlidingWindowOldEventsExpire(t *testing.T) {
	tracker := NewStormBudgetTracker(&StormBudgetConfig{
		MaxResourceStartsPerSecond: 2,
		WindowDuration:             50 * time.Millisecond,
	})

	// Fill the window
	tracker.CheckResource()
	tracker.CheckResource()

	// Should fail
	if err := tracker.CheckResource(); err != ErrBudgetExceeded {
		t.Errorf("Should be rate limited, got %v", err)
	}

	// Wait for first event to expire
	time.Sleep(60 * time.Millisecond)

	// Should succeed now
	if err := tracker.CheckResource(); err != nil {
		t.Errorf("After expiration should succeed, got %v", err)
	}
}
