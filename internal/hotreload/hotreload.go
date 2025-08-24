package hotreload

import (
	"context"
	"log/slog"
	"time"
)

// Manager manages the entire hot reload system
type Manager struct {
	watcher     *Watcher
	coordinator *Coordinator
	broadcaster *Broadcaster
	started     bool
}

// NewManager creates a new hot reload manager
func NewManager() (*Manager, error) {
	watcher, err := NewWatcher()
	if err != nil {
		return nil, err
	}

	coordinator := NewCoordinator(watcher)
	broadcaster := NewBroadcaster()

	return &Manager{
		watcher:     watcher,
		coordinator: coordinator,
		broadcaster: broadcaster,
	}, nil
}

// AddWatch adds a file or directory to watch
func (m *Manager) AddWatch(path string) error {
	return m.watcher.Add(path)
}

// RemoveWatch removes a file or directory from watch
func (m *Manager) RemoveWatch(path string) error {
	return m.watcher.Remove(path)
}

// RegisterReloadable registers a reloadable component
func (m *Manager) RegisterReloadable(reloadable Reloadable) error {
	return m.coordinator.Register(reloadable)
}

// AddListener adds an event listener
func (m *Manager) AddListener(name string, listener Listener) error {
	return m.broadcaster.AddListener(name, listener)
}

// RemoveListener removes an event listener
func (m *Manager) RemoveListener(name string) {
	m.broadcaster.RemoveListener(name)
}

// Start starts the hot reload system
func (m *Manager) Start() error {
	if m.started {
		return nil
	}

	// Start the coordinator which includes the watcher
	if err := m.coordinator.Start(); err != nil {
		return err
	}

	m.started = true
	slog.Info("Hot reload system started")
	return nil
}

// Stop stops the hot reload system
func (m *Manager) Stop() {
	if !m.started {
		return
	}

	m.coordinator.Stop()
	m.broadcaster.Close()
	m.started = false
	slog.Info("Hot reload system stopped")
}

// SetDebounceTime sets the debounce time for reload events
func (m *Manager) SetDebounceTime(d time.Duration) {
	m.coordinator.SetDebounceTime(d)
}

// IsRunning returns whether the hot reload system is running
func (m *Manager) IsRunning() bool {
	return m.started
}

// Shutdown gracefully shuts down the hot reload system
func (m *Manager) Shutdown(ctx context.Context) error {
	m.Stop()
	return nil
}
