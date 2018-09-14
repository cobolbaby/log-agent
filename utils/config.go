package utils

import (
	"github.com/astaxie/beego/config"
	"os"
)

const (
	DEFAULT_CONF_PATH = "./conf/logagent.ini"
)

var (
	iniCfg config.Configer
)

func ConfigMgr() config.Configer {
	if iniCfg != nil {
		return iniCfg
	}
	filename := os.Getenv("LOGAGENT_CONF_PATH")
	if filename == "" {
		filename = DEFAULT_CONF_PATH
	}
	var err error
	iniCfg, err = config.NewConfig("ini", filename)
	if err != nil {
		panic("Failed to Load configuration. Please make sure that the configuration exists")
	}

	// TODO::iniCfg如何转化为Map对象
	// listenFileStr := conf.String("listen_file")
	// fileSlice := strings.Split(listenFileStr, ",")
	// for _, item := range fileSlice {
	// 	filename := strings.TrimSpace(item)
	// 	if len(filename) == 0 {
	// 		continue
	// 	}
	// 	appConfig.ListenFile = append(appConfig.ListenFile, filename)
	// }

	// appConfig.ThreadNum = conf.DefaultInt("thread_num", 8)
	// appConfig.KafkaAddr = conf.String("kafka::addr")
	// appConfig.KafkaTopic = conf.String("kafka::topic")
	// appConfig.LogFile = conf.String("log::file")
	// appConfig.LogLevel = conf.String("log::level")
	// return

	// 如何实现热加载

	return iniCfg
}
