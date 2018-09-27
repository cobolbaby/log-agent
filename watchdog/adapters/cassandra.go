package watchdog

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/gocql/gocql"
	"io/ioutil"
)

var (
	cluster *gocql.ClusterConfig
)

type CassandraAdapter struct {
	Name   string
	Config *CassandraAdapterCfg
	logger watchdog.Logger
}

type CassandraAdapterCfg struct {
	Hosts     []string
	Keyspace  string
	TableName string
}

func (this *CassandraAdapter) SetLogger(logger watchdog.Logger) watchdog.WatchdogAdapter {
	this.logger = logger
	return this
}

func (this *CassandraAdapter) Handle(fi watchdog.FileMeta) error {
	session, _ := this.NewCluster().CreateSession()
	defer session.Close()

	// 针对超大文件执行过滤操作
	if fi.Size > 16*1024*1024 {
		return errors.New("[CassandraAdapter]仅处理小于16M的文件")
	}
	// TODO:依据过滤条件过滤文件
	// TODO:判断是否为压缩文件

	dataBytes, err := ioutil.ReadFile(fi.Filepath)
	if err != nil {
		return err
	}
	// 通过下标获取元素进行修改
	// TODO:Gzip压缩，且需要保证内存使用率问题
	fi.ChunkData = dataBytes
	fi.Checksum = fmt.Sprintf("%x", md5.Sum(dataBytes))

	/*
		| Name          | Type      |          Key | Desc                                                                                                                                                                                                           |
		| :------------ | :-------- | -----------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
		| file_time     | timestamp | PARTITON KEY | 使用**文件创建时间**作为文件时间，在机台时间同步的基础上，可以作为有意义的业务时间（测试时间）。针对特殊的机台，会使用**路径名中包含的时间**，或者**上传 Cassandra 的时间**作为代替；这个逻辑实现在 Agent 里。 |
		| folder        | text      |  PRIMARY KEY | 文件所在路径名，从 Agent 监视路径下看的**相对路径**名。                                                                                                                                                        |
		| pack          | text      |  PRIMARY KEY | **压缩包文件名**，目前只支持 zip；非压缩文件时为“”                                                                                                                                                             |
		| name          | text      |  PRIMARY KEY | **文件名**；压缩包时，为**压缩包内完整路径+文件名**                                                                                                                                                            |
		| size          | int       |              | 文件原始大小（byte）                                                                                                                                                                                           |
		| modify_time   | timestamp |              | 文件修改时间                                                                                                                                                                                                   |
		| upload_time   | timestamp |              | 上传 Cassandra 的时间                                                                                                                                                                                          |
		| content       | blob      |              | 文件内容，目前支持上限是 16MB                                                                                                                                                                                  |
		| compress      | boolean   |              | 存储的文件内容**是否 gzip 压缩**，对于原始压缩类（jpg/jpeg/gif/png/wmv/flv/zip/gz）之外的文件，都建议压缩存储。                                                                                                |
		| compress_size | int       |              | 文件压缩后大小（byte），非压缩可以置 null                                                                                                                                                                      |
		| checksum      | text      |              | 原始文件内容校验值，目前是 MD5。                                                                                                                                                                               |
		| host          | text      |              | 原始上传机台的机器名。                                                                                                                                                                                         |
		| reference     | text      |              | 文件内容外部存储路径。                                                                                                                                                                                         |
	*/

	// item := map
	// return this.Insert(item)
	// return this.BatchInsert()
	return nil
}

func (this *CassandraAdapter) Insert() error {
	// if err := session.Query(`
	//   INSERT INTO users (id, firstname, lastname, email, city, age) VALUES (?, ?, ?, ?, ?, ?)`,
	//   gocqlUuid, user.FirstName, user.LastName, user.Email, user.City, user.Age).Exec(); err != nil {
	//   errs = append(errs, err.Error())
	// }
	return nil
}

func (this *CassandraAdapter) BatchInsert() error {
	// // unlogged batch, 进行批量插入，最好是partition key 一致的情况
	// t := time.Now()
	// batch := session.NewBatch(gocql.UnloggedBatch)
	// for i := 0; i < 100; i++ {
	//     batch.Query(`INSERT INTO bigrow (rowname, iplist) VALUES (?,?)`, fmt.Sprintf("name_%d", i), fmt.Sprintf("ip_%d", i))
	// }
	// if err := session.ExecuteBatch(batch); err != nil {
	//     fmt.Println("execute batch:", err)
	// }
	// bt := time.Now().Sub(t).Nanoseconds()
	return nil
}

func (this *CassandraAdapter) Rollback(fi watchdog.FileMeta) error {
	return nil
}

func (this *CassandraAdapter) NewCluster() *gocql.ClusterConfig {
	if cluster != nil {
		return cluster
	}
	cluster = gocql.NewCluster(this.Config.Hosts...)
	cluster.Keyspace = this.Config.Keyspace
	cluster.Consistency = gocql.Quorum
	// cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 10}
	// 设置连接池的数量，默认是2个(针对每一个host，都建立起NumConns个连接)
	// ？连接池的建立在createSession之前，还是之后
	cluster.NumConns = 3
	return cluster
}

// Should ignore filenames generated by
// Emacs, Vim or SublimeText
// func shouldIgnoreFile(filename string) bool {
// 	for _, regex := range ignoredFilesRegExps {
// 		r, err := regexp.Compile(regex)
// 		if err != nil {
// 			panic("Could not compile the regex: " + regex)
// 		}
// 		if r.MatchString(filename) {
// 			return true
// 		} else {
// 			continue
// 		}
// 	}
// 	return false
// }

// checkIfWatchExt returns true if the name HasSuffix <watch_ext>.
// func checkIfWatchExt(name string) bool {
// 	for _, s := range cfg.WatchExts {
// 		if strings.HasSuffix(name, s) {
// 			return true
// 		}
// 	}
// 	return false
// }
