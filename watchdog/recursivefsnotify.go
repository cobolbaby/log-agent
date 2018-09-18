package watchdog

import (
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
)

type RecursiveWatcher struct {
	*fsnotify.Watcher
}

func NewRecursiveWatcher() (*RecursiveWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &RecursiveWatcher{watcher}, nil
}

func (w *RecursiveWatcher) RegCallback(callback func(queue []string)) error {
	for {
		select {
		case event := <-w.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				w.RecursiveAdd(event.Name)
				changeFiles := []string{event.Name}
				callback(changeFiles)
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				changeFiles := []string{event.Name}
				callback(changeFiles)
				continue
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				w.RecursiveRemove(event.Name)
				continue
			}
			// if event.Op&fsnotify.Write == fsnotify.Write {
			//  changeFiles := []string{event.Name}
			// 	callback(changeFiles)
			// }
		case err := <-w.Errors:
			return err
		}
	}
	return nil
}

func (w *RecursiveWatcher) RecursiveAdd(name string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		err := w.Add(name)
		if err != nil {
			return err
		}
		return nil
	}
	// 	for _, skipDir := range s.SkipDirs {
	// 		if skipDir == "" {
	// 			continue
	// 		}
	// 		if strings.HasPrefix(path, filepath.Join(s.WorkDir, skipDir)) {
	// 			return filepath.SkipDir
	// 		}
	// 	}
	// 	return nil
	// for _, p := range s.Observables {
	// 	if match(p, path) {
	// 		dir := filepath.Dir(path)
	// 		if _, ok := memo[dir]; !ok {
	// 			memo[dir] = struct{}{}
	// 			_ = watcher.Add(dir)
	// 		}
	// 		break
	// 	}
	// }
	filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			w.Add(path)
		}
		return nil
	})
	return nil
}

func (w *RecursiveWatcher) RecursiveRemove(name string) error {
	fi, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		err := w.Remove(name)
		if err != nil {
			return err
		}
		return nil
	}
	filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			w.Remove(path)
		}
		return nil
	})
	return nil
}