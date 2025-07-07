package timestate

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

// Monitor monitors states with TTL expiration and notifies via channel.
// Uses min-heap for efficient expiration checks and map for O(1) state access.
// Generic type T must be comparable for state change detection.
type Monitor[K, T comparable] struct {
	heap        items[K, T]       // Min-heap ordered by Expires
	items       map[K]*item[K, T] // Key-value storage
	mu          sync.Mutex        // Thread safety
	defaultTTL  time.Duration     // Default state lifetime
	checkTicker *time.Ticker      // Periodic checker
	expiredCh   chan<- K          // Expiration notifications
}

// New creates a Monitor instance.
//
// Parameters:
//   - checkInterval: how often to check expirations (e.g., 1*time.Second)
//   - defaultTTL: default state lifetime (e.g., 5*time.Minute)
//   - expiredCh: buffered channel for expiration notifications (e.g., make(chan string, 100))
func New[K, T comparable](
	checkInterval time.Duration,
	defaultTTL time.Duration,
	expiredCh chan<- K,
) *Monitor[K, T] {
	return &Monitor[K, T]{
		heap:        make(items[K, T], 0),
		items:       make(map[K]*item[K, T]),
		defaultTTL:  defaultTTL,
		checkTicker: time.NewTicker(checkInterval),
		expiredCh:   expiredCh,
	}
}

// Watch adds or updates a state only if the value changed.
// Uses defaultTTL for new states. Returns true if state was updated.
func (m *Monitor[K, T]) Watch(key K, value T) bool {
	return m.watch(key, value, m.defaultTTL)
}

// watch updates a state with custom TTL if the value changed.
// Returns true if state was added/modified, false if unchanged.
func (m *Monitor[K, T]) watch(key K, value T, ttl time.Duration) bool {
	expires := time.Now().Add(ttl)

	m.mu.Lock()
	defer m.mu.Unlock()

	if it, exists := m.items[key]; exists {
		if it.Value == value {
			return false // unchanged
		}

		it.Value = value
		it.Expires = expires
		it.removed = false

		heap.Fix(&m.heap, 0) // reorder heap

		return true
	}

	newItem := &item[K, T]{
		Key:     key,
		Value:   value,
		Expires: expires,
	}
	m.items[key] = newItem
	heap.Push(&m.heap, newItem)

	return true
}

// Get retrieves a state's value and expiration time.
// Returns zero values if state doesn't exist or was removed.
func (m *Monitor[K, T]) Get(key K) (value T, expires time.Time, exists bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if it, ok := m.items[key]; ok && !it.removed {
		return it.Value, it.Expires, true
	}

	return value, time.Time{}, false
}

// Remove removes a state without expiration notification.
func (m *Monitor[K, T]) Remove(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if it, exists := m.items[key]; exists {
		it.removed = true

		delete(m.items, key)
	}
}

// Start begins monitoring in a background goroutine.
// Stop by canceling the context.
func (m *Monitor[K, T]) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *Monitor[K, T]) run(ctx context.Context) {
	for {
		select {
		case <-m.checkTicker.C:
			m.checkExpirations()
		case <-ctx.Done():
			m.checkTicker.Stop()

			return
		}
	}
}

func (m *Monitor[K, T]) checkExpirations() {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for m.heap.Len() > 0 {
		it := m.heap[0]
		if it.Expires.After(now) {
			break
		}

		heap.Pop(&m.heap)

		if !it.removed && m.items[it.Key] == it {
			select {
			case m.expiredCh <- it.Key:
				delete(m.items, it.Key)
			default:
				heap.Push(&m.heap, it) // requeue if channel full
			}
		}
	}
}

// item represents a single tracked entity with expiration.
type item[K, T comparable] struct {
	Key     K         // Unique identifier for the item
	Value   T         // Current state value
	Expires time.Time // Expiration timestamp
	removed bool      // Soft-delete flag
}

// items is a min-heap of items ordered by expiration time.
type items[K, T comparable] []*item[K, T]

func (h items[K, T]) Len() int { return len(h) }
func (h items[K, T]) Less(i, j int) bool {
	return h[i].Expires.Before(h[j].Expires)
}

func (h items[K, T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *items[K, T]) Push(x any) {
	*h = append(*h, x.(*item[K, T])) //nolint:forcetypeassert
}

func (h *items[K, T]) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]

	return item
}

var _ heap.Interface = (*items[any, any])(nil)
