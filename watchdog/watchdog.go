package watchdog

import (
	"errors"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
}

type FileMeta struct {
	Filepath     string
	Dirname      string
	Filename     string
	Size         int64
	Ext          string
	ModifyTime   time.Time
	UploadTime   time.Time
	ChunkData    []byte
	ChunkNo      uint32
	ChunkSize    uint64
	Compress     bool
	CompressSize int64
	Checksum     string
	LastOp       fsnotify.Event
}

type WatchdogAdapter interface {
	Handle(changeFiles []FileMeta) error
	SetLogger(logger Logger) WatchdogAdapter
}

type Watchdog struct {
	logger     Logger
	rules      []string
	adapters   []WatchdogAdapter
	delayQueue chan fsnotify.Event
	debounce   time.Duration
	fsEventQ   []fsnotify.Event
}

func Create() *Watchdog {
	this := &Watchdog{
		delayQueue: make(chan fsnotify.Event),
		debounce:   3000 * time.Microsecond,
	}
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

func (this *Watchdog) AddHandler(adapter WatchdogAdapter) *Watchdog {
	this.adapters = append(this.adapters, adapter)
	return this
}

func (this *Watchdog) Run() {
	// 使用闭包函数优化Debounce函数的生成
	debounceHandle := this.debounce(3*time.Second, func(events []fsnotify.Event) {
		this.handle(events)
	})
	this.listen(func(changeEvent fsnotify.Event) {
		debounceHandle(changeEvent)
	})

	// TODO:升级改造
	done := make(chan bool)
	go this.handleDelayQueue()
	go this.listen()
	// m := &Monitor{
	// 	startTime: time.Now(),
	// 	data:      SystemInfo{},
	// }
	// m.start(&lp)
	// 如果done中还没放数据，那main挂起，直到放数据为止
	<-done
}

func (this *Watchdog) listen() error {
	watcher, err := NewRecursiveWatcher()
	if err != nil {
		this.logger.Error("[NewRecursiveWatcher]%s", err)
		return err
	}
	defer watcher.Close()

	go watcher.HandleFsEvent(func(changeEvent fsnotify.Event) {
		this.delayQueue <- changeEvent
	})

	for _, rule := range this.rules {
		this.logger.Info("Listen Path: %s", rule)
		err := watcher.RecursiveAdd(rule)
		if err != nil {
			this.logger.Error("[RecursiveAdd]%s", err)
			return err
		}
	}

	return nil
}

func (this *Watchdog) handleDelayQueue() {
	timer := time.NewTimer(this.debounce)
	var e fsnotify.Event
	for {
		select {
		case e = <-this.delayQueue:
			this.fsEventQ = append(this.fsEventQ, e)
			// this.handle()
			timer.Reset(interval)
		case <-timer.C:
			if len(this.fsEventQ) == 0 {
				break
			}
			this.handle(this.fsEventQ)
			// 重置处理队列
			this.fsEventQ = []fsnotify.Event{}
		}
	}
}

func (this *Watchdog) filterEvents(fileEvents []fsnotify.Event) []fsnotify.Event {
	var list []fsnotify.Event
	keys := make(map[string]bool)
	// TODO:倒序循环，确保list中维持一个最新的事件列表
	for _, entry := range fileEvents {
		if _, value := keys[entry.Name]; !value {
			keys[entry.Name] = true
			list = append(list, entry)
		}
	}
	return list
}

func (this *Watchdog) handle(fileEvents []fsnotify.Event) error {
	fileEvents = this.filterEvents(fileEvents)
	// 获取changeFiles的metadata
	changeFileMeta, err := this.getFileMeta(fileEvents)
	if err != nil {
		return err
	}
	return this.adapterHandle(changeFileMeta)
}

func (this *Watchdog) getFileMeta(eventQ []fsnotify.Event) ([]FileMeta, error) {
	var fileMetas []FileMeta
	for _, event := range eventQ {
		fileMeta, err := this.getOneFileMeta(event)
		if err != nil {
			return nil, err
		}
		fileMetas = append(fileMetas, *fileMeta)
	}
	return fileMetas, nil
}

func (this *Watchdog) getOneFileMeta(fileEvent fsnotify.Event) (*FileMeta, error) {
	fileInfo, err := os.Lstat(fileEvent.Name)
	if err != nil {
		return new(FileMeta), err
	}
	if fileInfo.IsDir() {
		return new(FileMeta), errors.New("[getOneFileMeta]仅处理文件，忽略目录")
	}

	dirName, fileName := filepath.Split(fileEvent.Name)
	dirName, err = filepath.Abs(dirName)
	if err != nil {
		return new(FileMeta), err
	}

	return &FileMeta{
		Filepath:   filepath.Join(dirName, fileName),
		Dirname:    dirName,
		Filename:   fileName,
		Ext:        filepath.Ext(fileName),
		Size:       fileInfo.Size(),
		ModifyTime: fileInfo.ModTime(),
		LastOp:     fileEvent,
	}, nil
}

func (this *Watchdog) adapterHandle(files []FileMeta) error {
	var wg sync.WaitGroup
	for _, Adapter := range this.adapters {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(apdater WatchdogAdapter) {
			defer wg.Done()
			apdater.SetLogger(this.logger).Handle(files)
		}(Adapter)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}
