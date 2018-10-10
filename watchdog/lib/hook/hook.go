package hook

import (
	"errors"
	"fmt"
	"reflect"
)

// 插件需支持Run方法
type Plugin interface {
	Run() error
}

type Hook struct {
	tags map[string][]Plugin
}

func NewHook() *Hook {
	return &Hook{
		tags: make(map[string][]Plugin),
	}
}

func (this *Hook) Import(tag string, plugins ...Plugin) {
	this.tags[tag] = append(this.tags[tag], plugins...)
}

func (this *Hook) Get(tag string) []Plugin {
	if _, ok := this.tags[tag]; !ok {
		return []Plugin{}
	}
	return this.tags[tag]
}

func (this *Hook) Trigger(tag string, params ...interface{}) error {
	if _, ok := this.tags[tag]; !ok {
		return nil
	}
	for _, plugin := range this.tags[tag] {
		err := this.exec(plugin, params...)
		if err != nil {
			// 如果异常则中断插件执行
			return err
		}
	}
	return nil
}

func (this *Hook) exec(plugin Plugin, args ...interface{}) error {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	f := reflect.ValueOf(plugin).MethodByName("Run")
	if !f.IsValid() {
		msg := fmt.Sprintf("struct %s does not have method Run", reflect.TypeOf(plugin))
		return errors.New(msg)
	}
	f.Call(inputs)
	return nil
}
