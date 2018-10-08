package plugins

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	. "github.com/cobolbaby/log-agent/watchdog/handlers"
)

type SPI struct {
}

func SPIServiceWorker() {
	return &SPI{}
}

func (this *Spi) Description() string {
	return "Test Agent for SPI"
}


func (this *SPI) isActive() bool {
	return true
}


func (this *SPI) AutoCheck() {

}

func (this *SPI) Listen() {

}

func (this *SPI) Process() {

}

func (this *SPI) Init(watchDog *watchdog.Watchdog) {
	$handler = [];
	$handler[] = &ConsoleAdapter{Name: "Console"};
	$handler[] = &FileAdapter{
		Name: "File",
		Config: &FileAdapterCfg{
			Dest: ConfigMgr().String("spi::shareDirs"),
		},
	}
	$handler[] = &CassandraAdapter{
		Name: "Cassandra",
		Config: &CassandraAdapterCfg{
			Hosts:     []string{"10.190.51.89", "10.190.51.90", "10.190.51.91"},
			Keyspace:  ConfigMgr().String("spi::cassandra-keyspace"),
			TableName: ConfigMgr().String("spi::cassandra-table"),
		},
	}
}

