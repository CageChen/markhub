// Package watcher monitors file system changes and broadcasts events via callbacks.
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/CageChen/markhub/internal/config"
	"github.com/fsnotify/fsnotify"
)

// EventType represents the type of file system event
type EventType int

// File system event types.
const (
	EventCreate EventType = iota
	EventWrite
	EventRemove
	EventRename
)

// Event represents a file system change event
type Event struct {
	Type EventType
	Path string
}

// Callback is a function called when file changes occur
type Callback func(Event)

// Watcher monitors file system changes in the markdown directory
type Watcher struct {
	watcher   *fsnotify.Watcher
	cfg       *config.Config
	callbacks []Callback
	mu        sync.RWMutex
	done      chan struct{}
}

// New creates a new file system watcher
func New(cfg *config.Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher: w,
		cfg:     cfg,
		done:    make(chan struct{}),
	}, nil
}

// OnChange registers a callback for file change events
func (w *Watcher) OnChange(cb Callback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, cb)
}

// Start begins watching all configured directories
func (w *Watcher) Start() error {
	// Watch all configured folders (skip git_ref folders — they read from the object database)
	for _, folder := range w.cfg.Folders {
		if folder.GitRef != "" {
			continue
		}
		err := filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Only watch directories
			if info.IsDir() && !w.cfg.IsExcluded(path) {
				if err := w.watcher.Add(path); err != nil {
					log.Printf("Warning: cannot watch %s: %v", path, err)
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: failed to walk folder %s: %v", folder.Path, err)
		}
	}

	go w.eventLoop()
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Skip excluded paths
	if w.cfg.IsExcluded(event.Name) {
		return
	}

	// Only process markdown files
	if !isDir(event.Name) && !w.cfg.IsMarkdownFile(event.Name) {
		return
	}

	var eventType EventType
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = EventCreate
		// If a new directory is created, watch it
		if isDir(event.Name) {
			_ = w.watcher.Add(event.Name)
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = EventWrite
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = EventRemove
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = EventRename
	default:
		return
	}

	e := Event{
		Type: eventType,
		Path: event.Name,
	}

	w.mu.RLock()
	callbacks := make([]Callback, len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, cb := range callbacks {
		cb(e)
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
