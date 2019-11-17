package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// Watch watches for file creation events in `dir`. It executes `handler` on each event.
func Watch(dir string, handler func(filename string)) (*fsnotify.Watcher, error) {
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
					log.Println("Got create event for", event.Name)
					handler(event.Name)
				case event.Op&fsnotify.Write != 0:
					log.Println("Got write event for", event.Name)
					handler(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("Got watch error", err)
					return
				}
			}
		}
	}()
	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}
	log.Println("Watching", dir)
	return watcher, nil
}
