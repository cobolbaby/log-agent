package handler

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"dc-agent-go/watchdog/lib/compress"
	"dc-agent-go/watchdog/lib/log"
	"fmt"
	"github.com/gocql/gocql"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MAX_FILE_SIZE = 16 * 1024 * 1024
)

var (
	connections = make(map[string]*gocql.Session)
)

type CassandraAdapter struct {
	Name     string
	Config   *CassandraAdapterCfg
	logger   *log.LogMgr
	Priority uint8
	Session  *gocql.Session
}

type CassandraAdapterCfg struct {
	Hosts     string
	Keyspace  string
	TableName string
}

func NewCassandraAdapter(Cfg *CassandraAdapterCfg) (WatchdogHandler, error) {
	self := &CassandraAdapter{
		Name:   "Cassandra",
		Config: Cfg,
	}

	if err := self.CreateSession(); err != nil {
		return nil, err
	}
	if err := self.CheckCassandraTable(); err != nil {
		return nil, err
	}

	return self, nil
}

func (this *CassandraAdapter) SetLogger(logger *log.LogMgr) {
	this.logger = logger
}

func (this *CassandraAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *CassandraAdapter) Handle(fi FileMeta) error {

	// 针对超大文件执行过滤操作
	if fi.Size > MAX_FILE_SIZE {
		this.logger.Warn("[CassandraAdapter] %s 文件大小超过16M", fi.Filepath)
		return nil
	}

	// 如果为压缩文件需要特殊处理
	switch fi.Ext {
	case ".zip":
		return this.uploadZipedFile(&fi)
	default:
		return this.uploadUnArchivedFile(&fi)
	}
}

func (this *CassandraAdapter) uploadUnArchivedFile(fi *FileMeta) error {
	var err error
	for i, attempts := 1, 3; i <= attempts; i++ {
		if fi.Content, err = ioutil.ReadFile(fi.Filepath); err == nil {
			break
		}
		if i < attempts {
			this.logger.Warn("[CassandraAdapter] ioutil.ReadFile error: %s, retry #%d", err, i)
			time.Sleep(time.Duration(i) * time.Second)
		}
	}
	if err != nil {
		/*
			e.g.
			1) File Handle Error: open D:\\I1000_testlog\\HP\\Matterhorn\\K2786401B\\board\\20181213181445__All.txt: The process cannot access the file because it is being used by another process.
		*/
		this.logger.Error("[CassandraAdapter] %s ioutil.ReadFile error, %s", fi.Filepath, err)
		return err
	}
	return this.upload(fi)
}

// GBK转化为UTF8
func GBKToUTF8(src string) (string, error) {
	I := bytes.NewReader([]byte(src))
	O := transform.NewReader(I, simplifiedchinese.GBK.NewDecoder())
	res, e := ioutil.ReadAll(O)
	if e != nil {
		return "", e
	}
	return string(res), nil
}

func (this *CassandraAdapter) uploadZipedFile(fi *FileMeta) error {
	if fi.Size == 0 {
		this.logger.Error("[CassandraAdapter] %s is not a valid zip", fi.Filepath)
		// TODO:预警
		return nil
		// 对于非正确格式的Zip包，采用常规方式进行上传
		// return this.uploadUnArchivedFile(fi)
	}

	// Open a zip archive for reading.
	r, err := zip.OpenReader(fi.Filepath)
	if err != nil {
		this.logger.Error("[CassandraAdapter] %s zip.OpenReader error, %s", fi.Filepath, err)
		return err
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Windows下压缩包中的简体中文采用GBK编码
		if !utf8.ValidString(f.Name) {
			f.Name, err = GBKToUTF8(f.Name)
			if err != nil {
				this.logger.Warn("[CassandraAdapter] %s is not a valid utf-8/gbk string", f.Name)
				return err
			}
		}

		file := &FileMeta{
			Filepath:   fi.Filepath,
			SubDir:     fi.SubDir,
			Pack:       fi.Filename,
			Filename:   f.Name,
			Ext:        strings.ToLower(filepath.Ext(f.Name)),
			Size:       f.FileInfo().Size(),
			CreateTime: fi.CreateTime,
			ModifyTime: fi.ModifyTime,
			LastOp:     fi.LastOp,
			Host:       fi.Host,
			FolderTime: fi.FolderTime,
		}

		rc, err := f.Open()
		if err != nil {
			this.logger.Error("[CassandraAdapter] %s f.Open error, %s", f.Name, err)
			return err
		}
		defer rc.Close()

		if file.Content, err = ioutil.ReadAll(rc); err != nil {
			this.logger.Error("[CassandraAdapter] %s ioutil.ReadAll error, %s", f.Name, err)
			return err
		}

		if err = this.upload(file); err != nil {
			return err
		}
	}
	return nil
}

func (this *CassandraAdapter) upload(fi *FileMeta) error {
	fi.Checksum = fmt.Sprintf("%x", md5.Sum(fi.Content))

	// 压缩需要保证内存使用率问题
	if compress.CheckIfCompressSize(fi.Size) && compress.CheckIfCompressExt(fi.Ext) {
		fi.Compress = true

		var err error
		fi.Content, err = compress.GzipContent(fi.Content)
		if err != nil {
			this.logger.Error("[CassandraAdapter] %s couldn't be compressed, %s", fi.Filepath, err)
			return err
		}

		fi.CompressSize = int64(len(fi.Content))

	} else {
		fi.Compress = false
		// fi.Content = fi.Content
		fi.CompressSize = fi.Size
	}

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

	return this.Insert(fi)
}

// 如果新增的记录主键已经存在，则更新历史记录
func (this *CassandraAdapter) Insert(item *FileMeta) error {
	// file_date -- 当前时区时间，该字段仅为了方便业务查询，不用细究正确性
	// file_time -- UTC时间

	q := this.Session.Query(`
		INSERT INTO `+this.Config.TableName+`
		(
			file_date,
			file_time,
			folder,
			pack,
			name,
			size,
			modify_time,
			content,
			compress,
			compress_size,
			checksum,
			host,
			folder_time,
			upload_time
		)
		VALUES
		(
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`,
		item.CreateTime.Format("2006-01-02"),
		item.CreateTime,
		item.SubDir,
		item.Pack,
		item.Filename,
		item.Size,
		item.ModifyTime,
		item.Content,
		item.Compress,
		item.CompressSize,
		item.Checksum,
		item.Host,
		item.FolderTime,
		time.Now())
	err := q.Exec()
	if err != nil {
		this.logger.Error("[CassandraAdapter] Table %s insert couldn't be exec, %s", this.Config.TableName, err)
		return err
	}
	this.logger.Debug("[CassandraAdapter] Upload %s", item.Filepath)
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

func (this *CassandraAdapter) CreateSession() error {

	key := strings.Join([]string{"cassandra", "keyspace", this.Config.Keyspace}, ":")
	if session, ok := connections[key]; ok {
		this.Session = session
		return nil
	}

	hosts := strings.Split(this.Config.Hosts, ",")
	cluster := gocql.NewCluster(hosts...)
	cluster.ProtoVersion = 4

	// The authenticator is needed if password authentication is
	// enabled for your Cassandra installation. If not, this can
	// be removed.
	// cluster.Authenticator = gocql.PasswordAuthenticator{
	// 	Username: "some_username",
	// 	Password: "some_password",
	// }

	// gocql requires the keyspace to be provided before the session is created.
	cluster.Keyspace = this.Config.Keyspace

	cluster.Consistency = gocql.Quorum

	// [fix]gocql: unable to create session: unable to setup connection:
	// gocql: no response received from cassandra within timeout period
	cluster.Timeout = 30 * time.Second
	// Default retry policy to use for queries (default: 0)
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{NumRetries: 3, Min: 1000 * time.Millisecond}
	// Default reconnection policy to use for reconnecting before trying to mark host as down
	cluster.ReconnectionPolicy = &gocql.ConstantReconnectionPolicy{MaxRetries: 3, Interval: 5 * time.Second}

	session, err := cluster.CreateSession()
	if err != nil {
		this.logger.Error("Could not connect to Cassandra Cluster: %s", err)
		return err
	}

	this.Session = session
	connections[key] = session
	return nil
}

// Check if the table already exists. Create if table does not exist
func (this *CassandraAdapter) CheckCassandraTable() error {
	keySpaceMeta, _ := this.Session.KeyspaceMetadata(this.Config.Keyspace)

	if _, exists := keySpaceMeta.Tables[this.Config.TableName]; exists == true {
		return nil
	}

	table := "CREATE TABLE " + this.Config.Keyspace + "." + this.Config.TableName + `(
			file_date date,
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
			folder_time timestamp,
			PRIMARY KEY (file_date, file_time, folder, pack, name)
		  ) WITH compaction = { 'class':'LeveledCompactionStrategy' };`

	if err := this.Session.Query(table).Consistency(gocql.All).RetryPolicy(nil).Exec(); err != nil {
		return err
	}

	// DIY: Update table with something if it already exist.

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
