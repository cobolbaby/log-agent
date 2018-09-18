package watchdog

import (
	"strings"
	"sync"
)

type FileHandler interface {
	Handle(changeFiles []string) error
}

type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
}

type Watchdog struct {
	logger   Logger
	rules    []string
	adapters []FileHandler
}

func Create() *Watchdog {
	this := new(Watchdog)
	return this
}

func (this *Watchdog) SetLogger(logger Logger) *Watchdog {
	this.logger = logger
	return this
}

func (this *Watchdog) SetRules(rule string) *Watchdog {
	// 将rules按照分隔符拆分，合并当前规则
	ruleSlice := strings.Split(rule, ",")
	this.rules = append(this.rules, ruleSlice...)
	return this
}

func (this *Watchdog) AddHandler(adapter FileHandler) *Watchdog {
	this.adapters = append(this.adapters, adapter)
	return this
}

func (this *Watchdog) Run() {
	this.listen(func(changeFiles []string) {
		if len(changeFiles) > 0 {
			this.handle(changeFiles)
		}
		// ...
	})
}

func (this *Watchdog) listen(callback func(queue []string)) error {
	watcher, err := NewRecursiveWatcher()
	if err != nil {
		this.logger.Error("[NewRecursiveWatcher]", err)
		return err
	}
	defer watcher.Close()

	// ...
	done := make(chan bool)
	go watcher.RegCallback(callback)
	for _, rule := range this.rules {
		this.logger.Info(rule)
		err := watcher.RecursiveAdd(rule)
		if err != nil {
			this.logger.Error("[RecursiveAdd]", err)
			return err
		}
	}
	<-done

	return nil
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
