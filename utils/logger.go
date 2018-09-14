package utils

import (
	"github.com/astaxie/beego/logs"
)

/*
	如何实现单例模式
*/
func LogMgr() *logs.BeeLogger {
	log := logs.NewLogger()
	log.SetLogger(logs.AdapterConsole)
	return log
}
