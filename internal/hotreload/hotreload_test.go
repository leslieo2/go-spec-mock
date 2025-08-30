package hotreload

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// mockReloadableForManager is a mock for integration testing the manager.
type mockReloadableForManager struct {
	name        string
	reloadCount atomic.Int32
}

func (m *mockReloadableForManager) Reload(ctx context.Context) error {
	m.reloadCount.Add(1)
	return nil
}

func (m *mockReloadableForManager) Name() string {
	return m.name
}

func (m *mockReloadableForManager) GetReloadCount() int32 {
	return m.reloadCount.Load()
}

// mockListenerForManager is a mock listener for integration testing.
type mockListenerForManager struct {
	callCount atomic.Int32
}

func (m *mockListenerForManager) Listen(ctx context.Context, event Event) error {
	m.callCount.Add(1)
	return nil
}

func (m *mockListenerForManager) GetCallCount() int32 {
	return m.callCount.Load()
}

func setupTestDirForManager(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cleanup := func() {
		os.RemoveAll(dir)
	}
	return dir, cleanup
}

func TestNewManager(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.watcher == nil || m.coordinator == nil || m.broadcaster == nil {
		t.Fatal("Manager components not initialized")
	}
	if m.IsRunning() {
		t.Error("Manager should not be running initially")
	}
}

func TestManager_StartStop(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if m.IsRunning() {
		t.Fatal("Manager should not be running before Start()")
	}

	err = m.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !m.IsRunning() {
		t.Fatal("Manager should be running after Start()")
	}
	if !m.coordinator.IsRunning() {
		t.Fatal("Coordinator should be running after manager Start()")
	}

	// Starting again should be a no-op and return nil
	err = m.Start()
	if err != nil {
		t.Fatalf("Starting a running manager should not produce an error, got: %v", err)
	}

	m.Stop()
	if m.IsRunning() {
		t.Fatal("Manager should not be running after Stop()")
	}
	if m.coordinator.IsRunning() {
		t.Fatal("Coordinator should not be running after manager Stop()")
	}

	// Stopping again should be a no-op
	m.Stop()
}

func TestManager_Integration_Reload(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	m.SetDebounceTime(50 * time.Millisecond)

	testDir, cleanup := setupTestDirForManager(t)
	defer cleanup()
	testFile := filepath.Join(testDir, "config.yaml")
	err = os.WriteFile(testFile, []byte("key: value"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Register a reloadable component
	reloadable := &mockReloadableForManager{name: "config-loader"}
	err = m.RegisterReloadable(reloadable)
	if err != nil {
		t.Fatalf("RegisterReloadable() failed: %v", err)
	}

	// Add a watch path
	err = m.AddWatch(testDir)
	if err != nil {
		t.Fatalf("AddWatch() failed: %v", err)
	}

	// Start the manager
	err = m.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer m.Stop()

	// Wait a moment for the watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the file to trigger a reload
	err = os.WriteFile(testFile, []byte("key: new-value"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for the debounce and reload to happen
	time.Sleep(100 * time.Millisecond)

	if count := reloadable.GetReloadCount(); count != 1 {
		t.Errorf("Expected reload count to be 1, but got %d", count)
	}

	// Test RemoveWatch
	err = m.RemoveWatch(testDir)
	if err != nil {
		t.Fatalf("RemoveWatch() failed: %v", err)
	}

	// Modify the file again
	err = os.WriteFile(testFile, []byte("key: final-value"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file again: %v", err)
	}

	// Wait to ensure no new reload is triggered
	time.Sleep(100 * time.Millisecond)

	if count := reloadable.GetReloadCount(); count != 1 {
		t.Errorf("Expected reload count to remain 1 after removing watch, but got %d", count)
	}
}

func TestManager_AddRemoveListener(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	listener := &mockListenerForManager{}
	err = m.AddListener("test-listener", listener.Listen)
	if err != nil {
		t.Fatalf("AddListener() failed: %v", err)
	}

	if !m.broadcaster.HasListener("test-listener") {
		t.Fatal("Listener was not added to broadcaster")
	}

	m.RemoveListener("test-listener")
	if m.broadcaster.HasListener("test-listener") {
		t.Fatal("Listener was not removed from broadcaster")
	}
}

func TestManager_Shutdown(t *testing.T) {
	m, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	if err := m.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !m.IsRunning() {
		t.Fatal("Manager should be running after Start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = m.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	if m.IsRunning() {
		t.Error("Manager should not be running after Shutdown")
	}
}
