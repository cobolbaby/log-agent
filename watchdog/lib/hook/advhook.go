package hook

import (
	"errors"
	"fmt"
	"reflect"
)

// 插件实现的钩子支持自定义，没有严格的限制
type AdvancePlugin interface {
	IsActive() bool
}

type AdvanceHook struct {
	plugins []AdvancePlugin
}

func NewAdvanceHook() *AdvanceHook {
	return &AdvanceHook{}
}

func (this *AdvanceHook) Import(plugins ...AdvancePlugin) {
	for _, plugin := range plugins {
		if !reflect.ValueOf(plugin).MethodByName("IsActive").IsValid() {
			continue
		}
		if !plugin.IsActive() {
			continue
		}
		this.plugins = append(this.plugins, plugin)
	}
}

func (this *AdvanceHook) Get(hook string) []AdvancePlugin {
	var res []AdvancePlugin
	for _, plugin := range this.plugins {
		if !reflect.ValueOf(plugin).MethodByName(hook).IsValid() {
			continue
		}
		res = append(res, plugin)
	}
	return res
}

func (this *AdvanceHook) Trigger(hook string, params ...interface{}) error {
	plugins := this.Get(hook)
	if len(plugins) == 0 {
		return nil
	}
	for _, plugin := range plugins {
		err := this.exec(plugin, hook, params...)
		if err != nil {
			// 如果异常则中断插件执行
			return err
		}
	}
	return nil
}

func (this *AdvanceHook) exec(plugin AdvancePlugin, hook string, args ...interface{}) error {
	f := reflect.ValueOf(plugin).MethodByName(hook)
	if !f.IsValid() {
		msg := fmt.Sprintf("struct %s does not have method %s", reflect.TypeOf(plugin), hook)
		return errors.New(msg)
	}
	if len(args) == 0 {
		f.Call(nil)
		return nil
	}
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	f.Call(inputs)
	return nil
}
