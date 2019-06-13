package handler

import (
	"archive/zip"
	"crypto/md5"
	"github.com/cobolbaby/log-agent/watchdog/lib/compress"
	"github.com/cobolbaby/log-agent/watchdog/lib/log"
	"encoding/hex"
	// "encoding/json"
	"encoding/binary"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/linkedin/goavro/v2"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	KafkaInstance sarama.SyncProducer
)

const (
	// name字段中不能包含字符"-",不然会报：Input schema is an invalid Avro schema
	recordSchemaJSON = `
	{
		"type": "record",
		"name": "dcagent_value",
		"fields": [
			{
				"name": "file_date",
				"type": "string"
			},
			{
				"name": "file_time",
				"type": "long"
			},
			{
				"name": "folder",
				"type": "string",
				"default": ""
			},
			{
				"name": "pack",
				"type": "string",
				"default": ""
			},
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "size",
				"type": "long"
			},
			{
				"name": "modify_time",
				"type": "long"
			},
			{
				"name": "content",
				"type": "string"
			},
			{
				"name": "compress",
				"type": "boolean"
			},
			{
				"name": "compress_size",
				"type": "long"
			},
			{
				"name": "checksum",
				"type": "string"
			},
			{
				"name": "host",
				"type": "string"
			},
			{
				"name": "folder_time",
				"type": "long"
			}
		]
	}
	`
)

type KafkaAdapter struct {
	Name     string
	Config   *KafkaAdapterCfg
	logger   *log.LogMgr
	Priority uint8
	producer sarama.SyncProducer
	codec    *goavro.Codec
}

type KafkaAdapterCfg struct {
	Brokers  string
	Topic    string
	SchemaID uint
}

func NewKafkaAdapter(Cfg *KafkaAdapterCfg) (WatchdogHandler, error) {
	self := &KafkaAdapter{
		Name:   "Cassandra",
		Config: Cfg,
	}

	if err := self.newSyncProducer(); err != nil {
		return nil, err
	}

	return self, nil
}

func (this *KafkaAdapter) newSyncProducer() error {

	if KafkaInstance != nil {
		this.producer = KafkaInstance
		return nil
	}

	config := sarama.NewConfig()
	config.Version = sarama.V2_0_1_0
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionNone
	config.Producer.MaxMessageBytes = 10000000
	config.Producer.Retry.Max = 10
	config.Producer.Retry.Backoff = 1000 * time.Millisecond
	// sarama.MaxRequestSize =

	client, err := sarama.NewSyncProducer(strings.Split(this.Config.Brokers, ","), config)
	if err != nil {
		return err
	}
	// defer client.Close()
	KafkaInstance = client
	this.producer = client

	codec, err := goavro.NewCodec(recordSchemaJSON)
	if err != nil {
		return err
	}
	this.codec = codec

	return nil
}

func (this *KafkaAdapter) SetLogger(logger *log.LogMgr) {
	this.logger = logger
}

func (this *KafkaAdapter) GetPriority() uint8 {
	return this.Priority
}

func (this *KafkaAdapter) Handle(fi FileMeta) error {

	// 针对超大文件执行过滤操作
	// if fi.Size > MAX_FILE_SIZE {
	// 	this.logger.Warn("[KafkaAdapter] %s 文件大小超过16M", fi.Filepath)
	// 	return nil
	// }

	// 如果为压缩文件需要特殊处理
	switch fi.Ext {
	case ".zip":
		return this.uploadZipedFile(&fi)
	default:
		return this.uploadUnArchivedFile(&fi)
	}
}

func (this *KafkaAdapter) uploadUnArchivedFile(fi *FileMeta) error {
	var err error
	for i, attempts := 1, 3; i <= attempts; i++ {
		if fi.Content, err = ioutil.ReadFile(fi.Filepath); err == nil {
			break
		}
		if i < attempts {
			this.logger.Warn("[KafkaAdapter] ioutil.ReadFile error: %s, retry #%d", err, i)
			time.Sleep(time.Duration(i) * time.Second)
		}
	}
	if err != nil {
		/*
			e.g.
			1) File Handle Error: open D:\\I1000_testlog\\HP\\Matterhorn\\K2786401B\\board\\20181213181445__All.txt: The process cannot access the file because it is being used by another process.
		*/
		this.logger.Error("[KafkaAdapter] %s ioutil.ReadFile error, %s", fi.Filepath, err)
		return err
	}
	return this.upload(fi)
}

func (this *KafkaAdapter) uploadZipedFile(fi *FileMeta) error {
	if fi.Size == 0 {
		this.logger.Error("[KafkaAdapter] %s is not a valid zip", fi.Filepath)
		// TODO:预警
		return nil
		// 对于非正确格式的Zip包，采用常规方式进行上传
		// return this.uploadUnArchivedFile(fi)
	}

	// TODO:完善失败重试机制
	// Open a zip archive for reading.
	r, err := zip.OpenReader(fi.Filepath)
	if err != nil {
		this.logger.Error("[KafkaAdapter] %s zip.OpenReader error, %s", fi.Filepath, err)
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
				this.logger.Warn("[KafkaAdapter] %s is not a valid utf-8/gbk string", f.Name)
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
			this.logger.Error("[KafkaAdapter] %s f.Open error, %s", f.Name, err)
			return err
		}
		defer rc.Close()

		if file.Content, err = ioutil.ReadAll(rc); err != nil {
			this.logger.Error("[KafkaAdapter] %s ioutil.ReadAll error, %s", f.Name, err)
			return err
		}

		if err = this.upload(file); err != nil {
			return err
		}
	}
	return nil
}

func (this *KafkaAdapter) upload(fi *FileMeta) error {
	fi.Checksum = fmt.Sprintf("%x", md5.Sum(fi.Content))

	// 压缩需要保证内存使用率问题
	if compress.CheckIfCompressSize(fi.Size) && compress.CheckIfCompressExt(fi.Ext) {
		fi.Compress = true

		var err error
		fi.Content, err = compress.GzipContent(fi.Content)
		if err != nil {
			this.logger.Error("[KafkaAdapter] %s couldn't be compressed, %s", fi.Filepath, err)
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
func (this *KafkaAdapter) Insert(fi *FileMeta) error {
	// file_date -- 当前时区时间-日期，该字段仅为方便业务查询
	// file_time -- 当前时区时间-日期+时间

	// 针对空文件content保留为空
	var ctx string
	if fi.Size > 0 {
		ctx = "0x" + hex.EncodeToString(fi.Content)
	} else {
		ctx = string(fi.Content)
	}

	// msgKey := &MsgKeyEncoder{
	// 	Path: fi.Filepath,
	// }
	// Ref: http://cassandra.apache.org/doc/latest/cql/json.html#json-encoding-of-cassandra-data-types
	// 传时间戳，不传Datetime了
	// msgVal := &MsgValueEncoder{
	// 	CreateDate: fi.CreateTime.Format("2006-01-02"),
	// 	// CreateTime:   fi.CreateTime.UnixNano().Format("2006-01-02T15:04:05.000-0700"),
	// 	CreateTime:   fi.CreateTime.UnixNano() / 1000000,
	// 	SubDir:       fi.SubDir,
	// 	Pack:         fi.Pack,
	// 	Filename:     fi.Filename,
	// 	Size:         fi.Size,
	// 	ModifyTime:   fi.ModifyTime.UnixNano() / 1000000,
	// 	Compress:     fi.Compress,
	// 	CompressSize: fi.CompressSize,
	// 	Checksum:     fi.Checksum,
	// 	Host:         fi.Host,
	// 	FolderTime:   fi.FolderTime.UnixNano() / 1000000,
	// 	Content:      ctx,
	// }
	// msg := &sarama.ProducerMessage{
	// 	Topic:     this.Config.Topic,
	// 	Key:       msgKey,
	// 	Value:     msgVal,
	// 	Timestamp: time.Now(),
	// }
	native := map[string]interface{}{
		"file_date":     fi.CreateTime.Format("2006-01-02"),
		"file_time":     fi.CreateTime.UnixNano() / 1000000,
		"folder":        fi.SubDir,
		"pack":          fi.Pack,
		"name":          fi.Filename,
		"size":          fi.Size,
		"modify_time":   fi.ModifyTime.UnixNano() / 1000000,
		"compress":      fi.Compress,
		"compress_size": fi.CompressSize,
		"checksum":      fi.Checksum,
		"host":          fi.Host,
		"folder_time":   fi.FolderTime.UnixNano() / 1000000,
		"content":       ctx,
	}
	// Convert native Go form to binary Avro data
	msgVal, err := this.codec.BinaryFromNative(nil, native)
	if err != nil {
		fmt.Println(err)
	}
	// 此处无数坑，因为Confluent schema registry正对Avro序列化规则有特殊要求，不光需要序列化具体的内容，还要附加上Schema ID以及Magic Byte
	// Ref: https://docs.confluent.io/current/schema-registry/serializer-formatter.html#wire-format
	var binaryMsg []byte
	// Confluent serialization format version number; currently always 0.
	binaryMsg = append(binaryMsg, byte(0))
	// 4-byte schema ID as returned by Schema Registry
	binarySchemaId := make([]byte, 4)
	binary.BigEndian.PutUint32(binarySchemaId, uint32(this.Config.SchemaID))
	binaryMsg = append(binaryMsg, binarySchemaId...)
	// Avro serialized data in Avro's binary encoding
	binaryMsg = append(binaryMsg, msgVal...)

	msg := &sarama.ProducerMessage{
		Topic:     this.Config.Topic,
		Key:       sarama.StringEncoder(fi.Filepath),
		Value:     sarama.ByteEncoder(binaryMsg),
		Timestamp: time.Now(),
	}
	if _, _, err := this.producer.SendMessage(msg); err != nil {
		return err
	}
	this.logger.Debug("[KafkaAdapter] Upload %s", fi.Filepath)
	return nil
}

func (this *KafkaAdapter) Rollback(fi FileMeta) error {
	return nil
}

// type MsgKeyEncoder struct {
// 	Path string `json:"path"`
// }

// func (k *MsgKeyEncoder) Encode() ([]byte, error) {
// 	return json.Marshal(k)
// }

// func (k *MsgKeyEncoder) Length() int {
// 	encoded, _ := json.Marshal(k)
// 	return len(encoded)
// }

// // MsgValueEncoder Need to implement sarama.Encoder interface
// type MsgValueEncoder struct {
// 	CreateDate   string `json:"file_date"`
// 	CreateTime   int64  `json:"file_time"`
// 	SubDir       string `json:"folder"`
// 	Pack         string `json:"pack"`
// 	Filename     string `json:"name"`
// 	Size         int64  `json:"size"`
// 	ModifyTime   int64  `json:"modify_time"`
// 	Content      string `json:"content"`
// 	Compress     bool   `json:"compress"`
// 	CompressSize int64  `json:"compress_size"`
// 	Checksum     string `json:"checksum"`
// 	Host         string `json:"host"`
// 	FolderTime   int64  `json:"folder_time"`
// }

// func (v *MsgValueEncoder) Encode() ([]byte, error) {
// 	return json.Marshal(v)
// }

// func (v *MsgValueEncoder) Length() int {
// 	encoded, _ := json.Marshal(v)
// 	return len(encoded)
// }
