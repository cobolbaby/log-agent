package plugins

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	. "github.com/cobolbaby/log-agent/watchdog/adapters"
)

func StartSPI(watchDog *watchdog.Watchdog) {
	watchDog.SetRules(ConfigMgr().String("spi::watchDirs"))
	// 同步至共享目录
	watchDog.AddHandler(&FileAdapter{
		Name: "File",
		Config: &FileAdapterCfg{
			Dest: ConfigMgr().String("spi::shareDirs"),
		},
	})
	// 备份
	watchDog.AddHandler(&FileAdapter{
		Name: "File",
		Config: &FileAdapterCfg{
			Dest: ConfigMgr().String("spi::backupDirs"),
		},
	})
	// TODO:字符串转切片
	watchDog.AddHandler(&CassandraAdapter{
		Name: "Cassandra",
		Config: &CassandraAdapterCfg{
			Hosts:     []string{"10.190.51.89", "10.190.51.90", "10.190.51.91"},
			Keyspace:  ConfigMgr().String("spi::cassandra-keyspace"),
			TableName: ConfigMgr().String("spi::cassandra-table"),
		},
	})
}
