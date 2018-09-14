package command

import (
	. "github.com/cobolbaby/log-agent/utils"
)

func Start() {
	LogMgr().Info("LogAgent Monitor Switch State: %s :)", ConfigMgr().String("agent::switch"))

	// 支持添加至系统开机启动脚本中

	// 连接消息总线，维持长连接(权限校验，心跳维持)

	// 获取最新的配置信息，与本地文件Merge

	// 获取需要监控的文件匹配规则

	// 添加文件处理器(订阅发布者模式)
	// Console/Kafka/Cassandra/Ceph

	// 启动监控程序
	// Gzip压缩文件
	// 调用文件处理方法(模板方法)
}
