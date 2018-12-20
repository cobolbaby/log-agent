package plugins

import (
	. "github.com/cobolbaby/log-agent/utils"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/cobolbaby/log-agent/watchdog/handler"
	"github.com/cobolbaby/log-agent/watchdog/lib/hook"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"log"
	"reflect"
	"strings"
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
	watchDog.Logger.Info(this.Name() + " AutoCheck")

	for _, k := range []string{"watch", "cassandra_keyspace", "cassandra_table"} {
		if this.Config.HasKey(k) && this.Config.Key(k).Value() != "" {
			continue
		}
		errmsg := fmt.Sprintf("No config %q in section %q", k, this.Name())
		return errors.New(errmsg)
	}
	return nil
}

// 确认文件是否需要处理，或者是否存在异常
func (this *DefaultPlugin) CheckFile(watchDog *watchdog.Watchdog, file *handler.FileMeta) error {
	if file.LastOp.Biz != this.Name() {
		return nil
	}

	// 扩展代码...

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
	watchDog.Logger.Info(this.Name() + " AutoInit")

	CassandraAdapter, err := handler.NewCassandraAdapter(&handler.CassandraAdapterCfg{
		Hosts:     this.Config.Key("cassandra_hosts").Value(),
		Keyspace:  this.Config.Key("cassandra_keyspace").Value(),
		TableName: this.Config.Key("cassandra_table").Value(),
	})
	if err != nil {
		return err
	}
	watchDog.
		SetRules(this.Name(), this.Config.Key("watch").Value()).
		AddHandler(this.Name(), CassandraAdapter)

	// 根据配置判断是否进行加载
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
		v.NewKey("cassandra_hosts", cfg.Section("CASSANDRA").Key("hosts").Value())
		plugin.SetAttr("BizName", v.Name()).SetAttr("Config", v)
		// 判断插件是否处于激活状态
		if !plugin.IsActive() {
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins
}
