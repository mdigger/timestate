package timestate_test

import (
	"testing"
	"time"

	"github.com/mdigger/timestate"
)

func TestBasicOperations(t *testing.T) {
	expiredCh := make(chan string, 10)
	monitor := timestate.New[string, int](time.Second, 5*time.Minute, expiredCh)
	ctx := t.Context()
	monitor.Start(ctx)

	// Test adding new state
	if !monitor.Watch("server1", 1) {
		t.Error("Expected true for new state")
	}

	// Test updating with same state
	if monitor.Watch("server1", 1) {
		t.Error("Expected false for unchanged state")
	}

	// Test updating with new state
	if !monitor.Watch("server1", 2) {
		t.Error("Expected true for changed state")
	}

	// Test getting it
	if val, _, exists := monitor.Get("server1"); !exists || val != 2 {
		t.Error("Failed to get current state")
	}

	// Test untrack
	monitor.Remove("server1")
	if _, _, exists := monitor.Get("server1"); exists {
		t.Error("State should be removed")
	}
}

func TestExpiration(t *testing.T) {
	expiredCh := make(chan string, 10)
	monitor := timestate.New[string, string](
		100*time.Millisecond, // faster checks for test
		200*time.Millisecond, // short TTL
		expiredCh,
	)
	ctx := t.Context()
	monitor.Start(ctx)

	monitor.Watch("temp", "value")

	select {
	case key := <-expiredCh:
		if key != "temp" {
			t.Errorf("Unexpected expired ID: %s", key)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("State did not expire as expected")
	}
}

func TestCustomIDType(t *testing.T) {
	type CustomID struct{ A, B string }
	expiredCh := make(chan CustomID, 5)
	monitor := timestate.New[CustomID, float64](time.Second, time.Minute, expiredCh)
	ctx := t.Context()
	monitor.Start(ctx)

	key := CustomID{A: "zone1", B: "server1"}
	monitor.Watch(key, 3.14)

	if val, _, exists := monitor.Get(key); !exists || val != 3.14 {
		t.Error("Failed with custom ID type")
	}
}
