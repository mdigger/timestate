package timestate_test

import (
	"context"
	"fmt"
	"time"

	"github.com/mdigger/timestate"
)

func ExampleMonitor() {
	// Create tracker with string IDs and int states
	expiredCh := make(chan string, 10)
	monitor := timestate.New[string, int](
		time.Second,
		5*time.Minute,
		expiredCh,
	)

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.Start(ctx)

	// Track states
	monitor.Watch("server1", 100)
	monitor.Watch("server2", 200)

	// Update state (resets TTL)
	monitor.Watch("server1", 150)

	// Handle expirations
	go func() {
		for id := range expiredCh {
			fmt.Printf("Expired: %s\n", id)
		}
	}()

	// Get current state
	if val, expires, exists := monitor.Get("server1"); exists {
		fmt.Printf("Server1: %d (expires %v)\n", val, expires.Format(time.Kitchen))
	}
}
