package utils

import (
	"github.com/astaxie/beego/config"
	. "github.com/cobolbaby/log-agent/utils"
	"os"
)

const (
	DEFAULT_CONF_PATH = "./conf/logagent.ini"
)

/*
	如何实现单例模式
*/
func ConfigMgr() config.Configer {
	filename := os.Getenv("LOGAGENT_CONF_PATH")
	if filename == "" {
		filename = DEFAULT_CONF_PATH
	}
	iniCfg, err := config.NewConfig("ini", filename)
	if err != nil {
		LogMgr().Error("Failed to Load configuration. ")
		panic("Failed to Load configuration. Please make sure that the configuration exists")
	}
	return iniCfg
}
