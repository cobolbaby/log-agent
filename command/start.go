// +build linux

package command

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	. "github.com/cobolbaby/log-agent/watchdog/adapters"
)

func Start() {
	// Recover防止程序挂掉
	// defer func() {
	// 	err := Recover()
	// 	if err != nil {
	// 		LogMgr().Error("Service Restart!!!")
	// 		return
	// 	}
	// }()
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
	// 获取需要监控的文件匹配规则
	watchDog := watchdog.Create().SetRules(ConfigMgr().String("agent::watchRules")).SetLogger(LogMgr())
	// Console/Kafka/Cassandra/Ceph
	// TODO:如何注入Json结构的配置信息
	watchDog.AddHandler(&ConsoleAdapter{
		Name: "Console",
	})
	watchDog.AddHandler(&CassandraAdapter{
		Name: "Cassandra",
		Config: &CassandraAdapterCfg{
			Hosts:     "10.190.51.89",
			Keyspace:  "dc_agent",
			TableName: "spi",
		},
	})
	watchDog.AddHandler(&FileAdapter{
		Name: "File",
	})
	// 启动监控程序
	// 调用文件处理方法(模板方法)
	watchDog.Run()
}
