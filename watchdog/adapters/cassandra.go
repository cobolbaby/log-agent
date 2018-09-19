package watchdog

import (
	"fmt"
	"time"
	"github.com/gocql/gocql"
)

var (
	cluster *gocql.ClusterConfig
)

type CassandraAdapter struct {
	Name 	string
	Config 	map[string][string]
}

func (this *CassandraAdapter) SetConfig(config) error {
	this.Config = config
	return this
}

func (this *CassandraAdapter) Handle(files []FileMeta) error {
	// time.Sleep(time.Second) // 停顿一秒
	fmt.Println(">", time.Now(), ">>", this.Name)
	
	// getConn
	session, _ := this.NewCluster().CreateSession()
	defer session.Close()

	for _, v := range files {
		dataBytes, err := ioutil.ReadFile(v.Filepath)
		if err != nil {
			return err
		}
		v.content := string(dataBytes)
	}

	


	// TODO::uploadFile
	sql := `INSERT INTO tweet (timeline, id, text) VALUES (?, ?, ?)`
	err := session.Query(sql, "me", gocql.TimeUUID(), "hello world").Exec()
	if err != nil {
		log.Fatal(err)
	}

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
	cluster := gocql.NewCluster("10.190.51.89", "10.190.51.90", "10.190.51.91")
	cluster.Keyspace = "dc_agent"
	cluster.Consistency = gocql.Quorum
	return cluster
}
