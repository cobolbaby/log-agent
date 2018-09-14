package watchdog

import (
// "github.com/astaxie/beego/logs"
)

var observers = []
var rules = []

func create() {

}

func addWatchRules(rules array) {
	rules := rules
}

func addHandler(observer WatchdogObserver) {
	observers[] = observer 
}

func run() {
	changeFiles := listen()

	for _, v := range observers {
		observer.handle(changeFiles)
	}
}
