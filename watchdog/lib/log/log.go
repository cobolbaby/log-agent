package log

import (
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

type LogMgr struct {
	logger *logrus.Logger
}

func NewLogMgr(logPath string) *LogMgr {

	logrusLogger := logrus.New()

	// 设置日志格式为json格式
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr，标准错误）
	// 日志消息输出可以是任意的io.writer类型
	logFile, err := rotatelogs.New(filepath.Join(logPath, "logagent-%Y%m%d.log"))
	if err != nil {
		logrusLogger.SetOutput(os.Stdout)
		logrusLogger.Error("Conldn't create 'logs' directory, please make sure the directory permissions :)")
	} else {
		// logrusLogger.SetOutput(logFile)
		// Ps: 奇葩的问题，如果将标准输出放在前面，程序以服务的方式运行的时候，日志不会输出至文件中
		logrusLogger.SetOutput(io.MultiWriter(logFile, os.Stdout))
		logrusLogger.Info("WatchDog Bootstrap ...")
		logrusLogger.Infof("Record log to %s", logFile.CurrentFileName())
	}

	// 设置日志级别为info以上
	logrusLogger.SetLevel(logrus.InfoLevel)
	// logrusLogger.SetLevel(logrus.WarnLevel)

	return &LogMgr{
		logger: logrusLogger,
	}
}

func (this *LogMgr) Fatal(format string, v ...interface{}) {
	this.logger.Fatalf(format, v...)
}

func (this *LogMgr) Error(format string, v ...interface{}) {
	this.logger.Errorf(format, v...)
}

func (this *LogMgr) Warn(format string, v ...interface{}) {
	this.logger.Warnf(format, v...)
}

func (this *LogMgr) Info(format string, v ...interface{}) {
	this.logger.Infof(format, v...)
}

func (this *LogMgr) Debug(format string, v ...interface{}) {
	this.logger.Debugf(format, v...)
}
