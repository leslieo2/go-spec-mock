package hotreload

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBroadcaster(t *testing.T) {
	b := NewBroadcaster()
	if b == nil {
		t.Fatal("NewBroadcaster returned nil")
	}
	if b.listeners == nil {
		t.Error("listeners map not initialized")
	}
	if b.ListenerCount() != 0 {
		t.Errorf("Expected 0 listeners, got %d", b.ListenerCount())
	}
}

func TestBroadcaster_AddListener(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	listener := func(ctx context.Context, event Event) error { return nil }

	// Test adding a new listener
	err := b.AddListener("test1", listener)
	if err != nil {
		t.Fatalf("Failed to add listener: %v", err)
	}
	if !b.HasListener("test1") {
		t.Error("Expected listener 'test1' to exist")
	}
	if b.ListenerCount() != 1 {
		t.Errorf("Expected 1 listener, got %d", b.ListenerCount())
	}

	// Test adding a listener with an existing name
	err = b.AddListener("test1", listener)
	if err == nil {
		t.Fatal("Expected error when adding listener with duplicate name, but got nil")
	}
}

func TestBroadcaster_RemoveListener(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	listener := func(ctx context.Context, event Event) error { return nil }
	if err := b.AddListener("test1", listener); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	// Test removing an existing listener
	b.RemoveListener("test1")
	if b.HasListener("test1") {
		t.Error("Expected listener 'test1' to be removed")
	}
	if b.ListenerCount() != 0 {
		t.Errorf("Expected 0 listeners, got %d", b.ListenerCount())
	}

	// Test removing a non-existent listener (should not panic)
	b.RemoveListener("nonexistent")
}

func TestBroadcaster_Broadcast(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	var counter1, counter2 atomic.Int32
	var wg sync.WaitGroup

	listener1 := func(ctx context.Context, event Event) error {
		defer wg.Done()
		counter1.Add(1)
		return nil
	}

	listener2 := func(ctx context.Context, event Event) error {
		defer wg.Done()
		counter2.Add(1)
		return nil
	}

	// Test broadcast with no listeners
	err := b.Broadcast(context.Background(), Event{})
	if err != nil {
		t.Fatalf("Broadcast with no listeners failed: %v", err)
	}

	// Add listeners
	if err := b.AddListener("listener1", listener1); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}
	if err := b.AddListener("listener2", listener2); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	// Test broadcast with multiple listeners
	wg.Add(2)
	err = b.Broadcast(context.Background(), Event{Path: "/test", Op: 1})
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}
	wg.Wait()

	if counter1.Load() != 1 {
		t.Errorf("Expected listener1 to be called once, got %d", counter1.Load())
	}
	if counter2.Load() != 1 {
		t.Errorf("Expected listener2 to be called once, got %d", counter2.Load())
	}
}

func TestBroadcaster_Broadcast_Error(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	expectedErr := errors.New("listener error")

	listener1 := func(ctx context.Context, event Event) error {
		return nil
	}
	listener2 := func(ctx context.Context, event Event) error {
		return expectedErr
	}

	if err := b.AddListener("ok_listener", listener1); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}
	if err := b.AddListener("error_listener", listener2); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	err := b.Broadcast(context.Background(), Event{})
	if err == nil {
		t.Fatal("Expected an error from broadcast, but got nil")
	}

	expectedErrMsg := "broadcast failed with 1 errors"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestBroadcaster_Close(t *testing.T) {
	b := NewBroadcaster()
	if err := b.AddListener("test", func(ctx context.Context, event Event) error { return nil }); err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	b.Close()

	if b.ListenerCount() != 0 {
		t.Errorf("Expected 0 listeners after Close, got %d", b.ListenerCount())
	}

	// Check if context is canceled
	select {
	case <-b.ctx.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context not canceled after Close")
	}
}

func TestBroadcaster_Concurrency(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently add listeners
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("listener-%d", i)
			if err := b.AddListener(name, func(ctx context.Context, event Event) error { return nil }); err != nil {
				t.Errorf("AddListener failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if b.ListenerCount() != numGoroutines {
		t.Fatalf("Expected %d listeners, got %d", numGoroutines, b.ListenerCount())
	}

	// Concurrently remove listeners
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("listener-%d", i)
			b.RemoveListener(name)
		}(i)
	}
	wg.Wait()

	if b.ListenerCount() != 0 {
		t.Fatalf("Expected 0 listeners, got %d", b.ListenerCount())
	}
}
