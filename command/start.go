package command

import (
	"github.com/cobolbaby/log-agent/plugins"
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
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

	watchDog := watchdog.NewWatchdog()
	watchDog.SetHost(ConfigMgr().String("agent::host"))
	watchDog.SetLogger(LogMgr())
	watchDog.LoadActivePlugins(plugins.SPIServiceWorker())
	// watchDog.LoadActivePlugins(plugins.SPIServiceWorker())
	// watchDog.LoadActivePlugins(plugins.SPIServiceWorker())
	// watchDog.LoadActivePlugins(plugins.SPIServiceWorker())
	watchDog.Run()
}
