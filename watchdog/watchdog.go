package watchdog

import (
	"strings"
	"sync"
	"errors"
	"path/filepath"
)

type FileMeta struct {
	Filepath      string
	Dirname        string
	Filename      string
	Size          int64
	Ext      	  string
	ModifyTime    int32 // ?
	UploadTime    int32 // ?
	ChunkData     text  // ?
	ChunkNo		  int32
	ChunkSize	  int64
	Compress      bool
	CompressSize  int64
	Checksum      string
}

type FileHandler interface {
	Handle(changeFiles []FileMeta) error
	SetConfig(config string) FileHandler
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
		this.logger.Info("Listen: ", rule)
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
	// 获取changeFiles的metadata
	changeFilesMeta := this.getFileMeta(changeFiles)
	return this.adapterHandle(changeFilesMeta)
}

func (this *Watchdog) getFilesMeta(files []string) []FileMeta {
	filesMeta := []FileMeta
	for _, fi := range files {
		filesMeta = append(filesMeta, this.getFileMeta(fi))
	}
	return filesMeta
}

func (this *Watchdog) getFileMeta(file string) (FileMeta, error) {
	fileInfo, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, errors.New("process only file")
	}
	dirName, fileName := filepath.Split(file)
	dirName = filepath.Abs(dirName)
	return &FileMeta{
		Filepath:       filepath.Join(dirName, fileName),
		Dirname:		dirName,
		Filename:      	fileName,
		Ext:			filepath.Ext(fileName),
		Size:          	fileInfo.Size(),
		ModifyTime:    	fileInfo.ModTime(),
	}
}

func (this *Watchdog) adapterHandle(files []FileMeta) error {
	var wg sync.WaitGroup
	for _, Adapter := range this.adapters {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(handler FileHandler, files []FileMeta) {
			defer wg.Done()
			handler.Handle(files)
		}(Adapter, files)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	return nil
}