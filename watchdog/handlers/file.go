package watchdog

import (
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/otiai10/copy"
	"path"
	"time"
)

type FileAdapter struct {
	Name   string
	Config *FileAdapterCfg
	logger watchdog.Logger
}

type FileAdapterCfg struct {
	Dest string
}

func (this *FileAdapter) SetLogger(logger watchdog.Logger) watchdog.WatchdogAdapter {
	this.logger = logger
	return this
}

func (this *FileAdapter) Handle(fi watchdog.FileMeta) error {
	// 拷贝文件至目标目录
	this.logger.Info("[FileAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	// TODO:子目录路径获取异常
	destPath := path.Join(this.Config.Dest, fi.Filename)
	err := copy.Copy(fi.Filepath, destPath)
	if err != nil {
		this.logger.Error("[FileAdapter] %s => %s", fi.Filepath, err)
		return err
	}
	this.logger.Info("[FileAdapter] %s => %s", fi.Filepath, destPath)
	return nil
}

func (this *FileAdapter) Rollback(fi watchdog.FileMeta) error {
	return nil
}
