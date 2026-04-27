package source

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
)

// FileSource is an event source that watches a file for changes
// When the file changes, it triggers reconciliation
type FileSource struct {
	BaseSource
	filePath string
	deviceID string
	path     string
	watcher  *fsnotify.Watcher
	done     chan struct{}
	wg       DoneWaitGroup
}

// NewFileSource creates a new file source that watches the given file
// for changes and triggers reconciliation for the given device/path
func NewFileSource(filePath, deviceID, path string) *FileSource {
	return &FileSource{
		filePath: filePath,
		deviceID: deviceID,
		path:     path,
		done:     make(chan struct{}),
	}
}

// Start implements Source interface
func (s *FileSource) Start(ctx context.Context, ctrl controller.Controller) error {
	s.controller = ctrl
	var err error

	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	s.done = make(chan struct{})
	s.wg.Add()

	go s.run(ctx)

	err = s.watcher.Add(s.filePath)
	if err != nil {
		s.watcher.Close()
		return err
	}

	return nil
}

func (s *FileSource) run(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if event.Name == s.filePath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Remove)) {
				// Debounce multiple rapid writes
				time.Sleep(100 * time.Millisecond)
				evt := predicate.Event{
					DeviceID: s.deviceID,
					Path:     s.path,
					Type:     predicate.UpdateEvent,
				}
				s.EnqueueEvent(evt)
			}
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue
			_ = err
		}
	}
}

// Stop implements Source interface
func (s *FileSource) Stop() {
	close(s.done)
	_ = s.watcher.Close()
	s.wg.Wait()
}
