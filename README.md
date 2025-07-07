# TimeState - State Monitoring Library

[![Go Reference](https://pkg.go.dev/badge/github.com/mdigger/timestate.svg)](https://pkg.go.dev/github.com/mdigger/timestate)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

TimeState is a Go library for tracking states with time-to-live (TTL) expiration. It provides:

- Generic state tracking with custom ID and state types
- Efficient expiration checking using min-heap
- Thread-safe operations
- Configurable check intervals
- Expiration notifications via channel

## Features

- **Generic Types**: Supports any comparable types for both IDs and states
- **Efficient**: O(1) lookups + O(log n) expiration checks
- **Thread-Safe**: Safe for concurrent use
- **Configurable**: Custom TTLs and check intervals
- **Reliable**: Exactly-once expiration notifications

## Installation

```bash
go get github.com/mdigger/timestate
```

## Usage

### Basic Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mdigger/timestate"
)

func main() {
	expiredCh := make(chan string, 10)
	monitor := timestate.New[string, int](
		time.Second,
		5*time.Minute,
		expiredCh,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.Start(ctx)

	// Track states
	monitor.Watch("server1", 100)
	monitor.Watch("server2", 200)

	// Handle expirations
	go func() {
		for id := range expiredCh {
			fmt.Printf("Expired: %s\n", id)
		}
	}()

	// Get current state
	if val, expires, exists := monitor.Get("server1"); exists {
		fmt.Printf("Value: %d, Expires: %v\n", val, expires)
	}
}
```

### Custom Types

```go
type DeviceID struct {
	Region string
	Serial int
}

// Create tracker with custom ID type
monitor := timestate.New[DeviceID, string](
	time.Second,
	time.Hour,
	make(chan DeviceID),
)

monitor.Watch(DeviceID{"EU", 1234}, "online")
```
