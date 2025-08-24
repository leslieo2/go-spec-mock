package hotreload

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Listener represents a callback function for reload events
type Listener func(ctx context.Context, event Event) error

// Broadcaster manages event broadcasting to listeners
type Broadcaster struct {
	listeners map[string]Listener
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

// NewBroadcaster creates a new event broadcaster
func NewBroadcaster() *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broadcaster{
		listeners: make(map[string]Listener),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddListener adds a listener with a unique name
func (b *Broadcaster) AddListener(name string, listener Listener) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.listeners[name]; exists {
		return fmt.Errorf("listener %s already exists", name)
	}

	b.listeners[name] = listener
	slog.Debug("Added event listener", "name", name)
	return nil
}

// RemoveListener removes a listener by name
func (b *Broadcaster) RemoveListener(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.listeners, name)
	slog.Debug("Removed event listener", "name", name)
}

// Broadcast sends an event to all registered listeners
func (b *Broadcaster) Broadcast(ctx context.Context, event Event) error {
	b.mu.RLock()
	listeners := make([]Listener, 0, len(b.listeners))
	for _, listener := range b.listeners {
		listeners = append(listeners, listener)
	}
	b.mu.RUnlock()

	if len(listeners) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(listeners))

	for name, listener := range b.listeners {
		wg.Add(1)
		go func(n string, l Listener) {
			defer wg.Done()
			if err := l(ctx, event); err != nil {
				errors <- fmt.Errorf("listener %s failed: %w", n, err)
			}
		}(name, listener)
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	var broadcastErrors []error
	for err := range errors {
		broadcastErrors = append(broadcastErrors, err)
	}

	if len(broadcastErrors) > 0 {
		return fmt.Errorf("broadcast failed with %d errors", len(broadcastErrors))
	}

	return nil
}

// Close stops the broadcaster and removes all listeners
func (b *Broadcaster) Close() {
	b.cancel()
	b.wg.Wait()

	b.mu.Lock()
	defer b.mu.Unlock()

	b.listeners = make(map[string]Listener)
	slog.Debug("Event broadcaster closed")
}

// ListenerCount returns the number of registered listeners
func (b *Broadcaster) ListenerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.listeners)
}

// HasListener checks if a listener with the given name exists
func (b *Broadcaster) HasListener(name string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.listeners[name]
	return exists
}
