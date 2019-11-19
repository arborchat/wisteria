package watch

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// Watch watches for file creation and write events in `dir`. It executes `handler` on each event,
// and also logs events and errors into the specified logger.
func Watch(dir string, logger *log.Logger, handler func(filename string)) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				switch {
				case event.Op&fsnotify.Create != 0:
					logger.Println("Got create event for", event.Name)
					handler(event.Name)
				case event.Op&fsnotify.Write != 0:
					logger.Println("Got write event for", event.Name)
					handler(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					logger.Println("Got watch error", err)
					return
				}
			}
		}
	}()
	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}
	logger.Println("Watching", dir)
	return watcher, nil
}
