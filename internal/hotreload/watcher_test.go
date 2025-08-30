package hotreload

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// setupTestDir creates a temporary directory for testing.
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "watcher_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cleanup := func() {
		os.RemoveAll(dir)
	}
	return dir, cleanup
}

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	if w == nil {
		t.Fatal("NewWatcher() returned nil")
	}
	defer w.Stop()

	if w.isWatching {
		t.Error("Watcher should not be watching initially")
	}
}

func TestWatcher_AddRemove(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Test Add
	err = w.Add(testDir)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	absPath, _ := filepath.Abs(testDir)
	if len(w.paths) != 1 || w.paths[0] != absPath {
		t.Errorf("Expected path '%s' to be added, got %v", absPath, w.paths)
	}

	// Test Remove
	err = w.Remove(testDir)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}
	if len(w.paths) != 0 {
		t.Errorf("Expected path to be removed, paths slice is now: %v", w.paths)
	}
}

func TestWatcher_Add_NonExistentPath(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	err = w.Add("non-existent-path-for-testing")
	if err == nil {
		t.Fatal("Expected error when adding non-existent path, but got nil")
	}
}

func TestWatcher_StartStop(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}

	if w.IsWatching() {
		t.Fatal("Watcher should not be running before Start()")
	}

	w.Start()
	if !w.IsWatching() {
		t.Fatal("Watcher should be running after Start()")
	}

	// Calling Start again should be a no-op
	w.Start()
	if !w.IsWatching() {
		t.Fatal("Watcher should still be running after second Start()")
	}

	w.Stop()
	if w.IsWatching() {
		t.Fatal("Watcher should not be running after Stop()")
	}

	// Calling Stop again should be a no-op
	w.Stop()
	if w.IsWatching() {
		t.Fatal("Watcher should still not be running after second Stop()")
	}
}

func TestWatcher_EventFlow(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}

	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = w.Add(testDir)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	w.Start()
	defer w.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedEvent Event
	go func() {
		defer wg.Done()
		select {
		case receivedEvent = <-w.Events():
		case <-time.After(2 * time.Second):
			t.Error("Timed out waiting for file event")
		}
	}()

	// Give the watcher a moment to start up
	time.Sleep(100 * time.Millisecond)

	// Trigger an event
	err = os.WriteFile(testFile, []byte("modified"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file to trigger event: %v", err)
	}

	wg.Wait()

	if receivedEvent.Path == "" {
		t.Fatal("Did not receive any event")
	}

	expectedPath, _ := filepath.Abs(testFile)
	if receivedEvent.Path != expectedPath {
		t.Errorf("Expected event for path '%s', got '%s'", expectedPath, receivedEvent.Path)
	}

	// fsnotify behavior can be tricky (e.g., WRITE or CHMOD). We accept either.
	if receivedEvent.Op&fsnotify.Write == 0 && receivedEvent.Op&fsnotify.Chmod == 0 {
		t.Errorf("Expected WRITE or CHMOD operation, got %s", receivedEvent.Op)
	}
}

func TestWatcher_shouldSkipEvent(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	testCases := []struct {
		path     string
		expected bool
	}{
		{"/path/to/file.txt", false},
		{"/path/to/file.tmp", true},
		{"/path/to/file.swp", true},
		{"/path/to/.hiddenfile", true},
		{"/path/to/~tempfile", true},
		{"regular.go", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			if got := w.shouldSkipEvent(tc.path); got != tc.expected {
				t.Errorf("shouldSkipEvent(%q) = %v; want %v", tc.path, got, tc.expected)
			}
		})
	}
}
