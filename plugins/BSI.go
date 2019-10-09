package plugins

import (
	"github.com/cobolbaby/log-agent/watchdog"
	"github.com/cobolbaby/log-agent/watchdog/handler"
	// "regexp"
	"strconv"
	"strings"
	"time"
)

var (
// LOUP/1395T2936101/2017-06-07/MBBIVS171700094_1W_1_2017-06-07_16_15_42_797/FLOWLOG.ZIP
// <family>/<model>/<date>/<test_id>
// BSIStandardLog = regexp.MustCompile(`(.*?)/(.*?)/(.*?)/(.*?)/.*`)
)

type BSI struct {
	DefaultPlugin
}

// ETL小工具
func (this *BSI) Transform(watchDog *watchdog.Watchdog, file *handler.FileMeta) error {
	if file.LastOp.Biz != this.Name() {
		return nil
	}
	watchDog.Logger.Debugf(this.Name() + " Transform")

	// 扩展代码...
	// file_date作为分区键，应该是相对稳定的，取文件相关的创建时间/修改时间/访问时间都不理想，
	// 否则某个目录下的文件换个日期重新创建一次即使内容没变也将产生新的记录，从而对后续解析、统计也将带来错误
	// e.g. LOUP/1395T2936101/2017-06-07/MBBIVS171700094_1W_1_2017-06-07_16_15_42_797/FLOWLOG.ZIP

	// if !BSIStandardLog.MatchString(file.SubDir) {
	// 	return nil
	// }
	// 查找第一个匹配结果及其分组字符串
	// matched := BSIStandardLog.FindStringSubmatch(src)
	// 0: LOUP/1395T2936101/2017-06-07/MBBIVS171700094_1W_1_2017-06-07_16_15_42_797/FLOWLOG.ZIP
	// 1: LOUP
	// 2: 1395T2936101
	// 3: 2017-06-07
	// 4: MBBIVS171700094_1W_1_2017-06-07_16_15_42_797
	// testID := matched[4]

	pathArray := strings.Split(file.SubDir, "/")
	if len(pathArray) != 4 && len(pathArray) != 5 {
		return nil
	}

	testID := pathArray[len(pathArray)-1]
	file.FolderTime = this.convert2Time(testID)
	// 考虑到BSI业务中压缩文件存在重复创建的情况，为了保证Cassandra中数据记录的唯一性，特将文件的创建时间设置为文件目录的时间
	file.CreateTime = file.FolderTime

	return nil
}

func (this *BSI) convert2Time(str string) time.Time {
	// MBBIVS171700094_1W_1_2017-06-07_16_15_42_797
	testTime := str[(len(str) - 23):]
	datetimeArray := strings.Split(testTime, "_")
	dateArray := strings.Split(datetimeArray[0], "-")

	year, _ := strconv.Atoi(dateArray[0])
	month, _ := strconv.Atoi(dateArray[1])
	day, _ := strconv.Atoi(dateArray[2])
	hour, _ := strconv.Atoi(datetimeArray[1])
	min, _ := strconv.Atoi(datetimeArray[2])
	sec, _ := strconv.Atoi(datetimeArray[3])
	nsec, _ := strconv.Atoi(datetimeArray[4])

	return time.Date(year, time.Month(month), day, hour, min, sec, nsec*1000000, time.Local)
}

func init() {
	Register(&BSI{})
}
