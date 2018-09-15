package watchdog

import (
	. "github.com/cobolbaby/log-agent/watchdog/adapters"
)

// Name for adapter with beego official support
const (
	AdapterConsole   = "Console"
	AdapterFile      = "File"
	AdapterCassandra = "Cassandra"
	// AdapterKafka     = "kafka"
)

type FileHandler interface {
	Handle(changeFiles []string) error
}

type Watchdog struct {
	rules    []string
	adapters []FileHandler
}

type operator func(queue []string)

func Create() *Watchdog {
	this := new(Watchdog)
	return this
}

func (this *Watchdog) SetRules(rules string) error {
	this.rules = append(this.rules, rules)
	return nil
}

func (this *Watchdog) AddHandler(adapterName string) error {
	var adapter FileHandler
	switch adapterName {
	case AdapterConsole:
		adapter = &Console{Name: AdapterConsole}
	case AdapterFile:
		adapter = &File{Name: AdapterFile}
	case AdapterCassandra:
		adapter = &Cassandra{Name: AdapterCassandra}
	default:
		return nil
	}
	this.adapters = append(this.adapters, adapter)
	return nil
}

func (this *Watchdog) Run() error {
	return this.listen(func(changeFiles []string) {
		this.handle(changeFiles)
		// ...
	})
}

func (this *Watchdog) listen(callback operator) error {
	// this.rules
	files := []string{"/opt/a", "/opt/b"}
	callback(files)
	return nil
}

func (this *Watchdog) handle(changeFiles []string) error {
	for _, Adapter := range this.adapters {
		Adapter.Handle(changeFiles)
	}
	return nil
}
