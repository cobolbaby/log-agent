package watchdog

import (
	"errors"
	"fmt"
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/gocql/gocql"
	"io/ioutil"
	"time"
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
	Hosts     string
	Keyspace  string
	TableName string
}

func (this *CassandraAdapter) SetLogger(logger watchdog.Logger) watchdog.WatchdogAdapter {
	this.logger = logger
	return this
}

func (this *CassandraAdapter) Handle(files []watchdog.FileMeta) error {
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)

	// getConn
	session, _ := this.NewCluster().CreateSession()
	defer session.Close()

	// 修改引用值
	for index, v := range files {
		// TODO:针对超大文件执行过滤操作
		if v.Size > 16*1024*1024 {
			return errors.New("[CassandraAdapter]仅处理小于16M的文件")
		}
		dataBytes, err := ioutil.ReadFile(v.Filepath)
		if err != nil {
			return err
		}
		// 通过下标获取元素进行修改
		// TODO:Gzip压缩
		files[index].ChunkData = dataBytes
	}

	// TODO:uploadFile
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

	// "folder",
	// "filename",
	// "size",
	// "time_modify",
	// "time_upload"
	// "content",
	// "compress",
	// "compress_size",
	// "checksum",
	// "date_test",
	// filetype
	// filepath
	// archived

	return nil
}

func (this *CassandraAdapter) NewCluster() *gocql.ClusterConfig {
	if cluster != nil {
		return cluster
	}
	cluster := gocql.NewCluster(this.Config.Hosts)
	cluster.Keyspace = this.Config.Keyspace
	cluster.Consistency = gocql.Three
	// 设置连接池的数量，默认是2个(针对每一个host，都建立起NumConns个连接)
	// ？连接池的建立在createSession之前，还是之后
	cluster.NumConns = 3
	return cluster
}
