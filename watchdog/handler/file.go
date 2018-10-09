package handler

import (
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"github.com/otiai10/copy"
	"path"
	"time"
)

type FileAdapter struct {
	Name     string
	Config   *FileAdapterCfg
	logger   log.Logger
	Priority int
}

type FileAdapterCfg struct {
	Dest string
}

func (this *FileAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *FileAdapter) Handle(fi FileMeta) error {
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

func (this *FileAdapter) Rollback(fi FileMeta) error {
	return nil
}
