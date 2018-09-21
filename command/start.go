// +build linux

package command

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	. "github.com/cobolbaby/log-agent/watchdog/adapters"
)

func Start() {
	// 连接消息总线，维持长连接
	// 获取主机唯一标示，用于辨识Agent
	// 订阅最新的配置信息

	agentSwitch, err := ConfigMgr().Bool("agent::switch")
	if err != nil {
		LogMgr().Error("undefined agent::switch")
		return
	}
	if !agentSwitch {
		LogMgr().Info("LogAgent Monitor Switch State: %s :)", ConfigMgr().String("agent::switch"))
	}

	watchDog := watchdog.Create()
	watchDog.SetLogger(LogMgr())
	watchDog.AddHandler(&ConsoleAdapter{Name: "Console"})

	// TODO:根据不同的业务获取不同的配置，同时调用不同的业务
	startSPIService(watchDog)
	// TODO:中间件实现，且明确如何做到反射
	// watchDog.Use(SPIServiceWorker())

	// 启动监控程序
	watchDog.Run()
}

func startSPIService(watchDog *watchdog.Watchdog) {

	watchDog.SetRules(ConfigMgr().String("agent::watchRules"))
	// 同步至共享目录
	watchDog.AddHandler(&FileAdapter{
		Name: "File",
		Config: &FileAdapterCfg{
			Dest: ConfigMgr().String("agent::watchRules"),
		},
	})
	// 备份
	watchDog.AddHandler(&FileAdapter{
		Name: "File",
		Config: &FileAdapterCfg{
			Dest: ConfigMgr().String("agent::watchRules"),
		},
	})
	watchDog.AddHandler(&CassandraAdapter{
		Name: "Cassandra",
		Config: &CassandraAdapterCfg{
			Hosts:     "10.190.51.89",
			Keyspace:  "dc_agent",
			TableName: "spi",
		},
	})

}
