package log

import (
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

type Logger interface {
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
	Warningf(string, ...interface{})
	Warnf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Fatal(...interface{})
	Error(...interface{})
	Warn(...interface{})
	Info(...interface{})
	Debug(...interface{})
}

func NewLogger(path string) Logger {

	logrusLogger := logrus.New()

	// 设置日志格式为json格式
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr，标准错误）
	// 日志消息输出可以是任意的io.writer类型
	logFile, err := rotatelogs.New(filepath.Join(path, "logagent-%Y%m%d.log"))
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

	return logrusLogger
}
