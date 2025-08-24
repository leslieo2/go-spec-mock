package hotreload

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Reloadable represents an interface that can be reloaded
type Reloadable interface {
	Reload(ctx context.Context) error
	Name() string
}

// Coordinator manages the hot reload process
type Coordinator struct {
	watcher      *Watcher
	reloadables  map[string]Reloadable
	eventChan    chan Event
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	debounceTime time.Duration
	wg           sync.WaitGroup
	isRunning    bool
}

// NewCoordinator creates a new reload coordinator
func NewCoordinator(watcher *Watcher) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Coordinator{
		watcher:      watcher,
		reloadables:  make(map[string]Reloadable),
		eventChan:    make(chan Event, 100),
		ctx:          ctx,
		cancel:       cancel,
		debounceTime: 500 * time.Millisecond,
		isRunning:    false,
	}
}

// Register adds a reloadable component to the coordinator
func (c *Coordinator) Register(reloadable Reloadable) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := reloadable.Name()
	if _, exists := c.reloadables[name]; exists {
		return fmt.Errorf("reloadable %s already registered", name)
	}

	c.reloadables[name] = reloadable
	slog.Info("Registered reloadable component", "name", name)
	return nil
}

// Unregister removes a reloadable component from the coordinator
func (c *Coordinator) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.reloadables, name)
	slog.Info("Unregistered reloadable component", "name", name)
}

// Start begins the hot reload coordination
func (c *Coordinator) Start() error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return fmt.Errorf("coordinator already running")
	}
	c.isRunning = true
	c.mu.Unlock()

	// Start the watcher
	c.watcher.Start()

	// Start event processing
	c.wg.Add(2)
	go c.processEvents()
	go c.coordinateReloads()

	slog.Info("Hot reload coordinator started")
	return nil
}

// Stop stops the hot reload coordination
func (c *Coordinator) Stop() {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = false
	c.mu.Unlock()

	c.cancel()
	c.watcher.Stop()
	close(c.eventChan)
	c.wg.Wait()

	slog.Info("Hot reload coordinator stopped")
}

// processEvents processes events from the watcher
func (c *Coordinator) processEvents() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case event, ok := <-c.watcher.Events():
			if !ok {
				return
			}
			select {
			case c.eventChan <- event:
			case <-c.ctx.Done():
				return
			}
		}
	}
}

// coordinateReloads coordinates the reload process with debouncing
func (c *Coordinator) coordinateReloads() {
	defer c.wg.Done()

	var (
		debounceTimer *time.Timer
		events        []Event
	)

	for {
		select {
		case <-c.ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event := <-c.eventChan:
			events = append(events, event)

			if debounceTimer == nil {
				debounceTimer = time.NewTimer(c.debounceTime)
			} else {
				debounceTimer.Reset(c.debounceTime)
			}

		case <-debounceTimer.C:
			if len(events) > 0 {
				c.triggerReload(events)
				events = events[:0] // Clear events slice
			}
			debounceTimer = nil
		}
	}
}

// triggerReload triggers the reload process for all registered reloadables
func (c *Coordinator) triggerReload(events []Event) {
	c.mu.RLock()
	reloadables := make([]Reloadable, 0, len(c.reloadables))
	for _, r := range c.reloadables {
		reloadables = append(reloadables, r)
	}
	c.mu.RUnlock()

	if len(reloadables) == 0 {
		return
	}

	slog.Info("Triggering hot reload", "events", len(events))

	// Log the triggering events
	for _, event := range events {
		slog.Debug("Reload triggered by", "path", event.Path, "operation", event.Op.String())
	}

	// Reload all components concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(reloadables))

	for _, reloadable := range reloadables {
		wg.Add(1)
		go func(r Reloadable) {
			defer wg.Done()
			if err := r.Reload(c.ctx); err != nil {
				errors <- fmt.Errorf("failed to reload %s: %w", r.Name(), err)
			} else {
				slog.Info("Successfully reloaded component", "name", r.Name())
			}
		}(reloadable)
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	var reloadErrors []error
	for err := range errors {
		reloadErrors = append(reloadErrors, err)
	}

	if len(reloadErrors) > 0 {
		slog.Error("Hot reload completed with errors", "errors", len(reloadErrors))
		for _, err := range reloadErrors {
			slog.Error("Reload error", "error", err)
		}
	} else {
		slog.Info("Hot reload completed successfully")
	}
}

// SetDebounceTime sets the debounce time for reload events
func (c *Coordinator) SetDebounceTime(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debounceTime = d
}

// IsRunning returns whether the coordinator is currently running
func (c *Coordinator) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}
