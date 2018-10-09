package hook

import (
	"errors"
	"fmt"
	"reflect"
)

type Service interface {
	AutoCheck() error
	Listen() error
}

type Hook struct {
	tags map[string][]Service
}

// 动态添加插件到某个标签
func (this *Hook) Import(tag string, name ...Service) {
	if _, ok := this.tags[tag]; !ok {
		this.tags[tag] = make([]Service, 0)
	}
	this.tags[tag] = append(this.tags[tag], name...)
}

func (this *Hook) Get(tag string) []Service {
	if _, ok := this.tags[tag]; !ok {
		return make([]Service, 0)
	}
	return this.tags[tag]
}

func (this *Hook) Listen(tag string, params ...interface{}) error {
	if _, ok := this.tags[tag]; !ok {
		return nil
	}
	// if(APP_DEBUG) {
	// 	G($tag.'Start');
	// 	trace('[ '.$tag.' ] --START--','','DEBUG');
	// }
	for _, v := range this.tags[tag] {
		err := this.exec(v, tag, params...)
		if err != nil {
			// 如果返回则中断插件执行
			return err
		}
	}
	// if(APP_DEBUG) { // 记录行为的执行日志
	// 	trace('[ '.$tag.' ] --END-- [ RunTime:'.G($tag.'Start',$tag.'End',6).'s ]','','DEBUG');
	//
	return nil
}

func (this *Hook) exec(object interface{}, methodName string, args ...interface{}) error {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	f := reflect.ValueOf(object).MethodByName(methodName)
	if !f.IsValid() {
		msg := fmt.Sprintf("struct %s does not have method %s", reflect.TypeOf(object), methodName)
		return errors.New(msg)
	}
	f.Call(inputs)
	return nil
}
