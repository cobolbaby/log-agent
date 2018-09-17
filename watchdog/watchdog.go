package watchdog

import (
	// "fmt"
	// "time"
	// "github.com/radovskyb/watcher"
	// "log"
	"sync"
)

type FileHandler interface {
	Handle(changeFiles []string) error
}

type Watchdog struct {
	rules    []string
	adapters []FileHandler
}

type operator func(queue []string)

func Create() *Watchdog {
	this := new(Watchdog)
	return this
}

func (this *Watchdog) SetRules(rules string) error {
	this.rules = append(this.rules, rules)
	return nil
}

func (this *Watchdog) AddHandler(adapter FileHandler) error {
	this.adapters = append(this.adapters, adapter)
	return nil
}

func (this *Watchdog) Run() {
	this.listen(func(changeFiles []string) {
		if len(changeFiles) > 0 {
			this.handle(changeFiles)
		}
		// ...
	})
}

func (this *Watchdog) listen(callback operator) {
	// // this.rules
	// w := watcher.New()

	// // SetMaxEvents to 1 to allow at most 1 event's to be received
	// // on the Event channel per watching cycle.
	// //
	// // If SetMaxEvents is not set, the default is to send all events.
	// w.SetMaxEvents(1)

	// // Only notify rename and move events.
	// w.FilterOps(watcher.Rename, watcher.Move)

	// go func() {
	// 	for {
	// 		select {
	// 		case event := <-w.Event:
	// 			fmt.Println(event) // Print the event's info.
	// 		case err := <-w.Error:
	// 			log.Fatalln(err)
	// 		case <-w.Closed:
	// 			return
	// 		}
	// 	}
	// }()

	// // Watch this folder for changes.
	// if err := w.AddRecursive("./test"); err != nil {
	// 	log.Fatalln(err)
	// }

	// // Watch test_folder recursively for changes.
	// // if err := w.AddRecursive("/tmp"); err != nil {
	// // 	log.Fatalln(err)
	// // }

	// // Print a list of all of the files and folders currently
	// // being watched and their paths.
	// for path, f := range w.WatchedFiles() {
	// 	fmt.Printf("%s: %s\n", path, f.Name())
	// }

	// fmt.Println()

	// // Trigger 2 events after watcher started.
	// go func() {
	// 	w.Wait()
	// 	w.TriggerEvent(watcher.Create, nil)
	// 	w.TriggerEvent(watcher.Remove, nil)
	// }()

	// // Start the watching process - it'll check for changes every 100ms.
	// if err := w.Start(time.Millisecond * 100); err != nil {
	// 	log.Fatalln(err)
	// }

	files := []string{"/opt/abc"}
	callback(files)
}

func (this *Watchdog) handle(changeFiles []string) error {
	var wg sync.WaitGroup
	for _, Adapter := range this.adapters {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(fileHandler FileHandler, files []string) {
			defer wg.Done()
			fileHandler.Handle(files)
		}(Adapter, changeFiles)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}
