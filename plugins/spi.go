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

func (this *SPI) bizName() string {
	return "SPI"
}

func (this *SPI) Description() string {
	return "DC-Agent For SPI"
}

func (this *SPI) IsActive() bool {
	return true
}

// TODO:检查配置文件中的配置是否正确
func (this *SPI) AutoCheck() error {
	fmt.Println("SPI AutoCheck")
	return nil

}

// TODO:确认文件是否需要处理，或者是否存在异常
func (this *SPI) CheckFile(file *handler.FileMeta) error {
	if file.LastOp.Biz != this.bizName() {
		return nil
	}
	return nil
}

// TODO:ETL小工具
func (this *SPI) Transform(file *handler.FileMeta) error {
	if file.LastOp.Biz != this.bizName() {
		return nil
	}
	return nil
}

// [必须]初始化
func (this *SPI) Init(watchDog *watchdog.Watchdog) {

	watchDog.
		SetRules(this.bizName(), ConfigMgr().String("spi::watchDirs")).
		// AddHandler((this.bizName(), &handler.FileAdapter{
		// 	Name: "File",
		// 	Config: &handler.FileAdapterCfg{
		// 		Dest: ConfigMgr().String("spi::shareDirs"),
		// 	},
		// }).
		// AddHandler(this.bizName(), &handler.CassandraAdapter{
		// 	Name: "Cassandra",
		// 	Config: &handler.CassandraAdapterCfg{
		// 		Hosts:     []string{"10.190.51.89", "10.190.51.90", "10.190.51.91"},
		// 		Keyspace:  ConfigMgr().String("spi::cassandra-keyspace"),
		// 		TableName: ConfigMgr().String("spi::cassandra-table"),
		// 	},
		// }).
		AddHandler(this.bizName(), &handler.ConsoleAdapter{
			Name: "Console",
		})

}
