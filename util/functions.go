package util

// import "fmt"

/*
 * 加载配置文件
 */
func LoadCfg(confType string, filename string) (err error) {
	// conf, err := config.NewConfig(confType, filename)
	// if err != nil {
	// 	fmt.Printf("Load the configuration error. [error=%v]\n", err)
	// 	return
	// }
	// appConfig = &model.Config{}

	// // log
	// appLogLevel, appLogPath := loadLogs(conf)

	// // collect
	// chanSize, err := loadCollect(conf)
	// if err != nil {
	// 	return
	// }

	// // kafka
	// kafkaAddress, err := loadKafkaConf(conf)
	// if err != nil {
	// 	return
	// }

	// // etcd
	// etcdAddress, etcdKey, err := loadEtcdConf(conf)
	// if err != nil {
	// 	return
	// }

	// appConfig.AppLogLevel = appLogLevel
	// appConfig.AppLogPath = appLogPath
	// appConfig.ChanSize = chanSize
	// appConfig.KafkaConf.Address = kafkaAddress
	// appConfig.EtcdConf.Address = etcdAddress
	// appConfig.EtcdConf.Key = etcdKey
	// fmt.Printf("Load conf finished,[AppConfig=%v]\n", appConfig)
	return
}

func ReloadCfg() {

}

func StoreCfg() {

}
