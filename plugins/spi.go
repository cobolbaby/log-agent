package plugins

import (
	"fmt"
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/cobolbaby/log-agent/watchdog/handler"
	"github.com/cobolbaby/log-agent/watchdog/watcher"
)

type SPI struct {
}

func SPIServiceWorker() *SPI {
	return &SPI{}
}

func (this *SPI) TagName() string {
	return "SPI"
}

func (this *SPI) Description() string {
	return "DC-Agent For SPI"
}

func (this *SPI) IsActive() bool {
	return true
}

func (this *SPI) AutoCheck() error {
	fmt.Println("SPI AutoCheck")
	// TODO:检查配置文件中的配置是否正确
	return nil

}

func (this *SPI) Listen() error {
	return nil
}

func (this *SPI) Process() error {
	return nil
}

func (this *SPI) Init(watchDog *watchdog.Watchdog) {

	watchDog.SetWatcher(this.TagName(), watcher.NewFsnotifyWatcher())
	watchDog.SetRules(this.TagName(), ConfigMgr().String("spi::watchDirs"))
	watchDog.AddHandler(this.TagName(), &handler.ConsoleAdapter{
		Name:     "Console",
		Priority: 1,
	})
	// watchDog.AddHandler(&handler.FileAdapter{
	// 	Name: "File",
	// 	Config: &handler.FileAdapterCfg{
	// 		Dest: ConfigMgr().String("spi::shareDirs"),
	// 	},
	//  	Priority: 0,
	// })
	watchDog.AddHandler(this.TagName(), &handler.CassandraAdapter{
		Name: "Cassandra",
		Config: &handler.CassandraAdapterCfg{
			Hosts:     []string{"10.190.51.89", "10.190.51.90", "10.190.51.91"},
			Keyspace:  ConfigMgr().String("spi::cassandra-keyspace"),
			TableName: ConfigMgr().String("spi::cassandra-table"),
		},
		Priority: 0,
	})

}
