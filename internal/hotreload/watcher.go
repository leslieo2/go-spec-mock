package hotreload

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher handles file system watching for hot reload
type Watcher struct {
	watcher    *fsnotify.Watcher
	paths      []string
	events     chan Event
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	isWatching bool
}

// Event represents a file system event
type Event struct {
	Path string
	Op   fsnotify.Op
}

// NewWatcher creates a new file watcher
func NewWatcher() (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Watcher{
		watcher:    fsWatcher,
		paths:      make([]string, 0),
		events:     make(chan Event, 100),
		ctx:        ctx,
		cancel:     cancel,
		isWatching: false,
	}, nil
}

// Add adds a file or directory to watch
func (w *Watcher) Add(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := w.watcher.Add(absPath); err != nil {
		return fmt.Errorf("failed to add path %s: %w", absPath, err)
	}

	w.paths = append(w.paths, absPath)
	slog.Debug("Added watch path", "path", absPath)
	return nil
}

// Remove removes a file or directory from watch
func (w *Watcher) Remove(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := w.watcher.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove path %s: %w", absPath, err)
	}

	// Remove from paths slice
	for i, p := range w.paths {
		if p == absPath {
			w.paths = append(w.paths[:i], w.paths[i+1:]...)
			break
		}
	}

	slog.Debug("Removed watch path", "path", absPath)
	return nil
}

// Events returns the channel for file system events
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Start begins watching for file system events
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.isWatching {
		w.mu.Unlock()
		return
	}
	w.isWatching = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.watch()
	slog.Info("File watcher started")
}

// Stop stops watching for file system events
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.isWatching {
		w.mu.Unlock()
		return
	}
	w.isWatching = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()
	close(w.events)
	if err := w.watcher.Close(); err != nil {
		slog.Error("Failed to close file watcher", "error", err)
	}
	slog.Info("File watcher stopped")
}

// watch is the main event loop for the watcher
func (w *Watcher) watch() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Skip temporary files and directories
			if w.shouldSkipEvent(event.Name) {
				continue
			}

			w.events <- Event{
				Path: event.Name,
				Op:   event.Op,
			}

			slog.Debug("File system event", "path", event.Name, "operation", event.Op.String())

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}

// shouldSkipEvent determines if an event should be skipped
func (w *Watcher) shouldSkipEvent(path string) bool {
	// Skip temporary files
	if filepath.Ext(path) == ".tmp" ||
		filepath.Ext(path) == ".swp" ||
		filepath.Base(path)[0] == '.' ||
		filepath.Base(path)[0] == '~' {
		return true
	}
	return false
}

// IsWatching returns whether the watcher is currently active
func (w *Watcher) IsWatching() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isWatching
}
