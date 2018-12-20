[TOC]

## FAQ 

### 安装依赖

```
$ dep ensure -add github.com/gocql/gocql
Fetching sources...

Solving failure: No versions of github.com/fsnotify/fsnotify met constraints:
	v1.4.7: unable to deduce repository and source type for "golang.org/x/sys/unix": unable to read metadata: go-import metadata not found
	v1.4.2: Could not introduce github.com/fsnotify/fsnotify@v1.4.2, as it is not allowed by constraint ^1.4.7 from project github.com/cobolbaby/log-agent.

```

### `cqlsh` 连接超时

```
$ cqlsh 10.190.51.89 
Connection error: ('Unable to connect to any servers', {'10.190.51.89': ProtocolError("cql_version '3.3.1' is not supported by remote (w/ native protocol). Supported versions: [u'3.4.4']",)})
$ cqlsh --version   
cqlsh 5.0.1
$ cqlsh --cqlversion 3.4.4 10.190.51.89
Connected to bigdatatest_cluster at 10.190.51.89:9042.
[cqlsh 5.0.1 | Cassandra 3.11.2 | CQL spec 3.4.4 | Native protocol v4]
Use HELP for help.

$ cqlsh 10.99.170.60     
Connection error: ('Unable to connect to any servers', {'10.99.170.60': error(None, "Tried connecting to [('10.99.170.60', 9042)]. Last error: timed out")})
$ cqlsh --cqlversion 3.4.4 --connect-timeout=10 10.99.170.60
```

### `Linux`环境下无法获取文件创建时间

```
$ stat a1
  文件：'a1'
  大小：34        	块：1          IO 块：4096   普通文件
设备：805h/2053d	Inode：106672      硬链接：1
权限：(0777/-rwxrwxrwx)  Uid：( 1000/cobolbaby)   Gid：( 1000/cobolbaby)
最近访问：2018-09-28 16:37:38.853810000 +0800
最近更改：2018-09-28 16:37:38.836991900 +0800
最近改动：2018-09-28 16:37:38.836991900 +0800
创建时间：-
```

### `Could not connect to Cassandra Cluster`

```
 gocql: unable to create session: unable to discover protocol version: dial tcp 10.190.51.91:9042: i/o timeout

 gocql: unable to create session: control: unable to setup connection: gocql: no response received from cassandra within timeout period
```

### `go: modules disabled inside GOPATH/src by GO111MODULE=auto`

```
$ go mod init
go: modules disabled inside GOPATH/src by GO111MODULE=auto; see 'go help modules'
$ export GO111MODULE=on
$ go mod init
```

### `cqlsh` 查询超时

```
$ cqlsh
OperationTimedOut: errors={'127.0.0.1': 'Client request timeout. See Session.execute[_async](timeout)'}, last_host=127.0.0.1
$ cqlsh --request-timeout=6000
```

### 创建表时一致性问题

```
IncomingTcpConnection.java:103 - UnknownColumnFamilyException reading from socket; closingorg.apache.cassandra.db.UnknownColumnFamilyException: Couldn't find table for cfId e976ce80-e23d-11e8-a179-bb51378f83a7. If a table was just created, this is likely due to the schema not being fully propagated.  Please wait for schema agreement on table creation

java.lang.RuntimeException: java.util.concurrent.ExecutionException: org.apache.cassandra.exceptions.ConfigurationException: Column family ID mismatch (found 39141ab0-e275-11e8-a179-bb51378f83a7; expected 37f2f340-e275-11e8-be01-73f35434cf3b)
```

遇到上述问题，直接`DROP`发生问题的`keyspace`，重新导入数据

### Cassandra GBK编码问题

### not a invalid zip file