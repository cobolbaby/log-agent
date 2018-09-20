### 目标支持以下功能

- 监控指定目录下文件变更

### Bugs 

- `GOPATH`

```
$ dep init
init failed: unable to detect the containing GOPATH: /opt/workspace/git/go-demo/src/github.com/cobolbaby/log-agent is not within any known GOPATH
```
通过`go env`查看`GOPATH`会发现`Go`程序获取的环境变量实属异常，尝试执行以下`export`命令看看

- 安装依赖

```
$ dep ensure -add github.com/gocql/gocql
Fetching sources...

Solving failure: No versions of github.com/fsnotify/fsnotify met constraints:
	v1.4.7: unable to deduce repository and source type for "golang.org/x/sys/unix": unable to read metadata: go-import metadata not found
	v1.4.2: Could not introduce github.com/fsnotify/fsnotify@v1.4.2, as it is not allowed by constraint ^1.4.7 from project github.com/cobolbaby/log-agent.

```

- `wrong type for SetLogger method`

```
/opt/workspace/git/go-demo/src/github.com/cobolbaby/log-agent/command/start.go:50:9: cannot use "github.com/cobolbaby/log-agent/watchdog/adapters".FileAdapter literal (type *"github.com/cobolbaby/log-agent/watchdog/adapters".FileAdapter) as type "github.com/cobolbaby/log-agent/watchdog".WatchdogAdapter in argument to watchDog.AddHandler:
	*"github.com/cobolbaby/log-agent/watchdog/adapters".FileAdapter does not implement "github.com/cobolbaby/log-agent/watchdog".WatchdogAdapter (wrong type for SetLogger method)
		have SetLogger("github.com/cobolbaby/log-agent/watchdog".Logger) *"github.com/cobolbaby/log-agent/watchdog/adapters".FileAdapter
		want SetLogger("github.com/cobolbaby/log-agent/watchdog".Logger) "github.com/cobolbaby/log-agent/watchdog".WatchdogAdapter
exit status 2
Process exiting with code: 1
```