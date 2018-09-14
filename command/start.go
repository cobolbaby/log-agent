package command

import (
	. "github.com/cobolbaby/log-agent/utils"
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

	// 获取最新的配置信息，与本地文件Merge

	// 添加文件处理器(订阅发布者模式)
	watchdog := Watchdog.create()

	// 获取需要监控的文件匹配规则
	watchdog.setRules(ConfigMgr().Bool("agent::monitored_rule"))

	// Console/Kafka/Cassandra/Ceph
	watchdog.addHandler("console")
	watchdog.addHandler("cassandra")
	watchdog.addHandler("file")

	// 启动监控程序
	// 调用文件处理方法(模板方法)
	watchdog.run()
}
