package handler

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"github.com/gocql/gocql"
	"io/ioutil"
	"time"
)

var (
// session *gocql.Session
)

type CassandraAdapter struct {
	Name     string
	Config   *CassandraAdapterCfg
	logger   log.Logger
	Priority uint8
}

type CassandraAdapterCfg struct {
	Hosts     []string
	Keyspace  string
	TableName string
}

func (this *CassandraAdapter) SetLogger(logger log.Logger) {
	this.logger = logger
}

func (this *CassandraAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *CassandraAdapter) Handle(fi FileMeta) error {
	this.logger.Info("[CassandraAdapter] -------------  %s  -------------", time.Now().Format("2006/1/2 15:04:05"))
	session, err := this.CreateSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 针对超大文件执行过滤操作
	if fi.Size > 16*1024*1024 {
		this.logger.Error("[CassandraAdapter] %s => 文件大小超过16M", fi.Filepath)
		return errors.New("[CassandraAdapter] 仅处理小于16M的文件")
	}
	// TODO:判断是否为压缩文件
	// TODO:Gzip压缩，且需要保证内存使用率问题

	dataBytes, err := ioutil.ReadFile(fi.Filepath)
	if err != nil {
		return err
	}
	fi.Pack = ""
	fi.Compress = false
	fi.CompressSize = 0
	fi.Content = dataBytes
	fi.Checksum = fmt.Sprintf("%x", md5.Sum(dataBytes))
	fi.BackUpTime = time.Now().UTC()

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

	return this.Insert(session, fi)
}

func (this *CassandraAdapter) Insert(session *gocql.Session, item FileMeta) error {
	// 如果新增的记录主键已经存在，则更新历史记录
	if err := session.Query(`
		INSERT INTO `+this.Config.TableName+`
		(
			file_time,
			folder,
			pack,
			name,
			size,
			modify_time,
			upload_time,
			content,
			compress,
			compress_size,
			checksum,
			host
		)
		VALUES
		(
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`,
		item.CreateTime,
		item.Dirname,
		item.Pack,
		item.Filename,
		item.Size,
		item.ModifyTime,
		item.BackUpTime,
		item.Content,
		item.Compress,
		item.CompressSize,
		item.Checksum,
		item.Host).Exec(); err != nil {
		this.logger.Error("[CassandraAdapter] %s => %s", item.Filepath, err)
		return err
	}
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

func (this *CassandraAdapter) Rollback(fi FileMeta) error {
	return nil
}

// TODO:如何保证协程共享数据库连接
func (this *CassandraAdapter) CreateSession() (*gocql.Session, error) {
	cluster := gocql.NewCluster(this.Config.Hosts...)

	// The authenticator is needed if password authentication is
	// enabled for your Cassandra installation. If not, this can
	// be removed.
	// cluster.Authenticator = gocql.PasswordAuthenticator{
	// 	Username: "some_username",
	// 	Password: "some_password",
	// }

	cluster.Keyspace = this.Config.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 2}

	// [fix]gocql: unable to create session: unable to setup connection:
	// gocql: no response received from cassandra within timeout period
	cluster.Timeout = 1 * time.Second
	// cluster.ProtoVersion = 4

	session, err := cluster.CreateSession()
	if err != nil {
		this.logger.Error("Could not connect to Cassandra Cluster: %s", err)
		return new(gocql.Session), err
	}

	// this.CheckCassandraTable(session)

	return session, nil
}

// Check if the table already exists. Create if table does not exist
func (this *CassandraAdapter) CheckCassandraTable(session *gocql.Session) error {
	keySpaceMeta, _ := session.KeyspaceMetadata(this.Config.Keyspace)

	if _, exists := keySpaceMeta.Tables[this.Config.TableName]; exists != true {
		session.Query(`
		CREATE TABLE ` + this.Config.Keyspace + "." + this.Config.TableName + ` (
			file_time timestamp,
			folder text,
			pack text,
			name text,
			size int,
			modify_time timestamp,
			upload_time timestamp,
			content blob,
			compress boolean,
			compress_size int,
			checksum text,
			host text,
			reference text,
			PRIMARY KEY (file_time, folder, pack, name)
		  ) WITH compaction = { 'class':'LeveledCompactionStrategy' };
		  `).Exec()
	}
	return nil
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
