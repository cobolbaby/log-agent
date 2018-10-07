package command

import (
	"github.com/cobolbaby/log-agent/plugins"
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	. "github.com/cobolbaby/log-agent/watchdog/handlers"
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
		return
	}

	// TODO:开启自检

	watchDog := watchdog.Create()
	watchDog.SetHost(ConfigMgr().String("agent::host"))
	watchDog.SetLogger(LogMgr())
	watchDog.AddHandler(&ConsoleAdapter{Name: "Console"})

	// TODO:中间件实现，且明确如何做到反射
	watchDog.Use(plugins.SPIServiceWorker())

	// 启动监控程序
	watchDog.Run()

}
