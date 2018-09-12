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
$ dep ensure -add github.com/radovskyb/watcher
Fetching sources...
"github.com/radovskyb/watcher" is not imported by your project, and has been temporarily added to Gopkg.lock and vendor/.
If you run "dep ensure" again before actually importing it, it will disappear from Gopkg.lock and vendor/.

```