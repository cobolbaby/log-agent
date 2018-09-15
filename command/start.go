package command

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
)

func Start() {
	agentSwitch, err := ConfigMgr().Bool("agent::switch")
	if err != nil {
		LogMgr().Error("undefined agent::switch")
		return
	}
	if !agentSwitch {
		LogMgr().Info("LogAgent Monitor Switch State: %s :)", ConfigMgr().String("agent::switch"))
	}

	// 连接消息总线，维持长连接(权限校验，心跳维持)

	// 订阅最新的配置信息

	// 添加文件处理器(订阅发布者模式)
	Watchdog := watchdog.Create()

	// 获取需要监控的文件匹配规则
	Watchdog.SetRules(ConfigMgr().String("agent::watchRules"))

	// Console/Kafka/Cassandra/Ceph
	Watchdog.AddHandler(watchdog.AdapterConsole)
	Watchdog.AddHandler(watchdog.AdapterCassandra)
	Watchdog.AddHandler(watchdog.AdapterFile)

	// 启动监控程序
	// 调用文件处理方法(模板方法)
	Watchdog.Run()
}
