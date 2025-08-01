package geoip

import (
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	reader   *SafeReader
	watcher  *fsnotify.Watcher
	stopChan chan struct{}
	logger   Logger
}

type Logger interface {
	Printf(format string, args ...interface{})
}

func NewWatcher(reader *SafeReader, logger Logger) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		reader:   reader,
		watcher:  watcher,
		stopChan: make(chan struct{}),
		logger:   logger,
	}, nil
}

func (w *Watcher) Start() error {
	err := w.watcher.Add(w.reader.Path())
	if err != nil {
		return err
	}

	go w.watchLoop()
	return nil
}

func (w *Watcher) Stop() {
	close(w.stopChan)
	w.watcher.Close()
}

func (w *Watcher) watchLoop() {
	var debounceTimer *time.Timer
	var mu sync.Mutex

	for {
		select {
		case <-w.stopChan:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				mu.Lock()
				// Annuler le timer précédent s'il existe
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Créer un nouveau timer avec debounce de 2 secondes
				debounceTimer = time.AfterFunc(2*time.Second, func() {
					w.logger.Printf("File %s modified, waiting for completion...", event.Name)

					// Attendre que le fichier soit stable (pas de modifications récentes)
					if w.waitForFileStability(w.reader.Path(), 5*time.Second) {
						w.logger.Printf("File stable, reloading GeoIP database...")
						if err := w.reader.Reload(); err != nil {
							w.logger.Printf("Failed to reload GeoIP database: %v", err)
						} else {
							w.logger.Printf("GeoIP database reloaded from %s", w.reader.Path())
						}
					} else {
						w.logger.Printf("File %s still unstable after timeout, skipping reload", w.reader.Path())
					}
				})
				mu.Unlock()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Printf("File watcher error: %v", err)
		}
	}
}

// waitForFileStability attend que le fichier ne soit plus modifié pendant une durée donnée
func (w *Watcher) waitForFileStability(filepath string, timeout time.Duration) bool {
	start := time.Now()
	var lastModTime time.Time

	for time.Since(start) < timeout {
		if stat, err := os.Stat(filepath); err == nil {
			currentModTime := stat.ModTime()
			if !lastModTime.IsZero() && currentModTime.Equal(lastModTime) {
				// Le fichier n'a pas été modifié depuis la dernière vérification
				if time.Since(currentModTime) > 1*time.Second {
					return true // Le fichier est stable
				}
			}
			lastModTime = currentModTime
		}
		time.Sleep(500 * time.Millisecond)
	}

	return false // Timeout atteint, fichier toujours instable
}
