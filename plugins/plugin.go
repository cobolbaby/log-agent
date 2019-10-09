package plugins

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/cobolbaby/log-agent/watchdog/handler"
	"github.com/cobolbaby/log-agent/watchdog/lib/fsnotify"
	"github.com/cobolbaby/log-agent/watchdog/lib/hook"
	"github.com/cobolbaby/log-agent/watchdog/watcher"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var (
	cfg     = ConfigMgr()
	structs = make(map[string]reflect.Type)
)

type Plugin interface {
	SetAttr(string, interface{}) Plugin
	Name() string
	IsActive() bool
	AutoCheck(*watchdog.Watchdog) error
	AutoInit(*watchdog.Watchdog) error
	Mount(*watchdog.Watchdog) error
	CheckFile(*watchdog.Watchdog, *handler.FileMeta) error
	Transform(*watchdog.Watchdog, *handler.FileMeta) error
	Handle404Error(watchDog *watchdog.Watchdog, file *handler.FileMeta, fevent *fsnotify.Event) error
}

type DefaultPlugin struct {
	BizName     string
	Description string
	Config      *ini.Section
}

func (this *DefaultPlugin) SetAttr(attr string, val interface{}) Plugin {
	switch attr {
	case "BizName":
		this.BizName = val.(string)
	case "Config":
		this.Config = val.(*ini.Section)
	}
	return this
}

func (this *DefaultPlugin) Name() string {
	return this.BizName
}

func (this *DefaultPlugin) IsActive() bool {
	if this.Config.HasKey("switch") &&
		this.Config.Key("switch").MustBool() == false {
		return false
	}
	// TODO:支持动态启用指定插件
	return true
}

// 检查配置文件中的配置是否正确
func (this *DefaultPlugin) AutoCheck(watchDog *watchdog.Watchdog) error {
	watchDog.Logger.Infof(this.Name() + " AutoCheck")

	// for _, k := range []string{"watch", "cassandra_keyspace", "cassandra_table"} {
	// for _, k := range []string{"watch", "kafka_brokers", "kafka_topic"} {
	for _, k := range []string{"watch"} {
		if this.Config.HasKey(k) && this.Config.Key(k).Value() != "" {
			continue
		}
		errmsg := fmt.Sprintf("No config %q in section %q", k, this.Name())
		return errors.New(errmsg)
	}
	return nil
}

// 确认文件是否需要处理，或者是否存在异常
// 即便错误也不应该影响程序执行，需要将错误抛出
func (this *DefaultPlugin) CheckFile(watchDog *watchdog.Watchdog, file *handler.FileMeta) error {
	if file.LastOp.Biz != this.Name() {
		return nil
	}

	// 扩展代码...
	// 检查即时生成文件的创建时间，确认机器时间设置是否正确
	// 如果时间区间偏差太大，则延迟抛出问
	// 如何通过检查的结果控制后续流程是否执行
	if file.LastOp.Op != "LOAD" {
		return nil
	}

	return nil
}

// ETL小工具
func (this *DefaultPlugin) Transform(watchDog *watchdog.Watchdog, file *handler.FileMeta) error {
	if file.LastOp.Biz != this.Name() {
		return nil
	}

	// 扩展代码...

	return nil
}

// 自动初始化
func (this *DefaultPlugin) AutoInit(watchDog *watchdog.Watchdog) error {
	watchDog.Logger.Infof(this.Name() + " AutoInit")

	watchDog.SetRules(this.Name(), &fsnotify.Rule{
		Biz:             this.Name(),
		RootPath:        this.Config.Key("watch").Value(),
		MonitPath:       filepath.Join(this.Config.Key("watch").Value(), this.Config.Key("subdir").Value()),
		Patterns:        this.Config.Key("patterns").Value(),
		Ignores:         this.Config.Key("ignores").Value(),
		MaxNestingLevel: this.Config.Key("max_nesting_level").MustUint(0),
		DebounceTime:    time.Duration(this.Config.Key("debounce").MustUint(3000)) * time.Millisecond, // 文件系统事件延迟处理时间. 每种业务的处理机制是不一样的, 可以设置一个默认的, 然后也可以针对单一业务做配置覆盖.
	})

	watchStrategy := []string{watcher.FS_NOTIFY}
	if !this.Config.HasKey("history_import") || (this.Config.HasKey("history_import") && this.Config.Key("history_import").MustBool() == true) {
		watchStrategy = append(watchStrategy, watcher.FS_POLL)
	}
	watchDog.SetWatchStrategy(this.Name(), watchStrategy)

	// 历史版本直接上传Cassandra
	if this.Config.HasKey("cassandra_hosts") && this.Config.HasKey("cassandra_keyspace") && this.Config.HasKey("cassandra_table") {
		CassandraAdapter, err := handler.NewCassandraAdapter(&handler.CassandraAdapterCfg{
			Hosts:     this.Config.Key("cassandra_hosts").Value(),
			Keyspace:  this.Config.Key("cassandra_keyspace").Value(),
			TableName: this.Config.Key("cassandra_table").Value(),
		})
		if err != nil {
			return err
		}
		watchDog.AddHandler(this.Name(), CassandraAdapter)
	}

	// 新版本先上传至Kafka
	if this.Config.HasKey("kafka_brokers") && this.Config.HasKey("kafka_topic") {
		KafkaAdapter, err := handler.NewKafkaAdapter(&handler.KafkaAdapterCfg{
			Brokers:        this.Config.Key("kafka_brokers").Value(),
			Topic:          this.Config.Key("kafka_topic").Value(),
			SchemaRegistry: this.Config.Key("kafka_schema_registry").Value(),
		})
		if err != nil {
			return err
		}
		watchDog.AddHandler(this.Name(), KafkaAdapter)
	}

	// 本地备份操作
	if this.Config.HasKey("backup") && this.Config.Key("backup").Value() != "" {
		FileAdapter, _ := handler.NewFileAdapter(&handler.FileAdapterCfg{
			DestRoot: this.Config.Key("backup").Value(),
		})
		watchDog.AddHandler(this.Name(), FileAdapter)
	}

	// TODO:连接消息总线，维持长连接

	return nil
}

// 添加业务特殊处理(同步目录)
func (this *DefaultPlugin) Mount(watchDog *watchdog.Watchdog) error {

	// 扩展代码...

	return nil
}

func (this *DefaultPlugin) Handle404Error(watchDog *watchdog.Watchdog, file *handler.FileMeta, fevent *fsnotify.Event) error {

	// 扩展代码...

	return nil
}

// Ref: https://play.golang.org/p/igmssSD9k2
func Register(plugin Plugin) {
	// 动态获取结构体名称
	t := reflect.TypeOf(plugin).Elem()
	structs[t.Name()] = t
}

func Autoload() []hook.AdvancePlugin {
	var plugins []hook.AdvancePlugin

	for _, v := range cfg.Sections() {
		if !v.HasKey("watch") {
			continue
		}
		name := strings.ToUpper(strings.Split(v.Name(), ".")[0])
		// 还是得研究一下反射那块
		t, ok := structs[name]
		if !ok {
			log.Fatalf("Plugin %q not yet exists :)", name)
		}
		plugin := reflect.New(t).Interface().(Plugin)
		// 动态设置配置信息
		// 历史版本直接上传Cassandra
		v.NewKey("cassandra_hosts", cfg.Section("CASSANDRA").Key("hosts").Value())
		// 新版本先上传至Kafka
		v.NewKey("kafka_brokers", cfg.Section("KAFKA").Key("brokers").Value())
		v.NewKey("kafka_schema_registry", cfg.Section("KAFKA").Key("schema_registry").Value())
		plugin.SetAttr("BizName", v.Name()).SetAttr("Config", v)
		// 判断插件是否处于激活状态
		if !plugin.IsActive() {
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins
}
