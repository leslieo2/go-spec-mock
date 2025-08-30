package hotreload

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// mockReloadable is a mock implementation of the Reloadable interface for testing.
type mockReloadable struct {
	name        string
	reloadCount atomic.Int32
	reloadFunc  func(ctx context.Context) error
	reloadDelay time.Duration
}

func (m *mockReloadable) Reload(ctx context.Context) error {
	if m.reloadDelay > 0 {
		time.Sleep(m.reloadDelay)
	}
	m.reloadCount.Add(1)
	if m.reloadFunc != nil {
		return m.reloadFunc(ctx)
	}
	return nil
}

func (m *mockReloadable) Name() string {
	return m.name
}

func (m *mockReloadable) GetReloadCount() int32 {
	return m.reloadCount.Load()
}

func TestNewCoordinator(t *testing.T) {
	w, _ := NewWatcher()
	defer w.Stop()
	c := NewCoordinator(w)
	if c == nil {
		t.Fatal("NewCoordinator returned nil")
	}
	if c.isRunning {
		t.Error("Coordinator should not be running initially")
	}
}

func TestCoordinator_RegisterUnregister(t *testing.T) {
	w, _ := NewWatcher()
	defer w.Stop()
	c := NewCoordinator(w)

	reloadable1 := &mockReloadable{name: "comp1"}

	// Test Register
	err := c.Register(reloadable1)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if _, ok := c.reloadables["comp1"]; !ok {
		t.Fatal("Component 'comp1' not found after registration")
	}

	// Test duplicate registration
	err = c.Register(reloadable1)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, but got nil")
	}

	// Test Unregister
	c.Unregister("comp1")
	if _, ok := c.reloadables["comp1"]; ok {
		t.Fatal("Component 'comp1' found after unregistration")
	}
}

func TestCoordinator_StartStop(t *testing.T) {
	w, _ := NewWatcher()
	c := NewCoordinator(w)

	if c.IsRunning() {
		t.Fatal("Coordinator should not be running before Start()")
	}
	if w.IsWatching() {
		t.Fatal("Watcher should not be running before coordinator Start()")
	}

	err := c.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !c.IsRunning() {
		t.Fatal("Coordinator should be running after Start()")
	}
	if !w.IsWatching() {
		t.Fatal("Watcher should be running after coordinator Start()")
	}

	// Test starting again
	err = c.Start()
	if err == nil {
		t.Fatal("Expected error when starting a running coordinator, but got nil")
	}

	c.Stop()
	if c.IsRunning() {
		t.Fatal("Coordinator should not be running after Stop()")
	}
	if w.IsWatching() {
		t.Fatal("Watcher should not be running after coordinator Stop()")
	}
}

func TestCoordinator_Debouncing(t *testing.T) {
	w, _ := NewWatcher()
	c := NewCoordinator(w)
	c.SetDebounceTime(50 * time.Millisecond)

	reloadable := &mockReloadable{name: "test"}
	if err := c.Register(reloadable); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	err := c.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer c.Stop()

	// Send multiple events within the debounce window
	c.eventChan <- Event{}
	c.eventChan <- Event{}
	c.eventChan <- Event{}

	time.Sleep(25 * time.Millisecond)
	if count := reloadable.GetReloadCount(); count != 0 {
		t.Fatalf("Reload triggered prematurely, count: %d", count)
	}

	// Wait for debounce timer to fire
	time.Sleep(75 * time.Millisecond)

	if count := reloadable.GetReloadCount(); count != 1 {
		t.Fatalf("Expected reload count to be 1, got %d", count)
	}

	// Send another event
	c.eventChan <- Event{}
	time.Sleep(75 * time.Millisecond)

	if count := reloadable.GetReloadCount(); count != 2 {
		t.Fatalf("Expected reload count to be 2, got %d", count)
	}
}

func TestCoordinator_TriggerReload(t *testing.T) {
	w, _ := NewWatcher()
	c := NewCoordinator(w)

	reloadable1 := &mockReloadable{name: "comp1"}
	reloadable2 := &mockReloadable{name: "comp2", reloadFunc: func(ctx context.Context) error {
		return errors.New("reload failed")
	}}

	if err := c.Register(reloadable1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := c.Register(reloadable2); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Use a temporary file to check logging output
	logFile, _ := os.CreateTemp("", "log")
	defer os.Remove(logFile.Name())
	// Note: In a real scenario, you'd redirect slog output. For this test, we'll just check for errors.

	c.triggerReload([]Event{{Path: "test.file"}})

	if reloadable1.GetReloadCount() != 1 {
		t.Errorf("Expected comp1 to be reloaded once, got %d", reloadable1.GetReloadCount())
	}
	if reloadable2.GetReloadCount() != 1 {
		t.Errorf("Expected comp2 to be reloaded once, got %d", reloadable2.GetReloadCount())
	}
	// We can't easily check the slog output without more setup, but we know the error path was taken.
}

func TestCoordinator_ContextCancellation(t *testing.T) {
	w, _ := NewWatcher()
	c := NewCoordinator(w)
	c.SetDebounceTime(100 * time.Millisecond)

	var ctxCancelled atomic.Bool
	reloadable := &mockReloadable{
		name: "long-reload",
		reloadFunc: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				ctxCancelled.Store(true)
				return ctx.Err()
			case <-time.After(300 * time.Millisecond):
				return nil
			}
		},
	}
	if err := c.Register(reloadable); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Trigger a reload
	c.eventChan <- Event{}

	// Give it time to start the reload
	time.Sleep(150 * time.Millisecond)

	// Stop the coordinator, which should cancel the context
	c.Stop()

	if !ctxCancelled.Load() {
		t.Error("Expected context to be canceled during reload, but it wasn't")
	}
}
