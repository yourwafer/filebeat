package service

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
	"xai.com/shushu/app/model"
)

var tasks sync.Map
var ticker *time.Ticker
var stopChan chan struct{}

type logTask struct {
	logPosition *logPosition
	rootPath    string
	relatePath  string
	port        string
	stop        chan struct{}
}

func (t *logTask) Close() {
	t.stop <- struct{}{}
	close(t.stop)
}

func (t *logTask) Closed() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}

func RegisterEvent(systemConfig *model.AppConfig, serverConfig *model.ServerConfig, recordName, logType string) bool {
	position := LoadOne(serverConfig.Operator, serverConfig.Server, recordName)
	if position != nil {
		_, ok := tasks.Load(position.Id)
		if ok {
			return false
		}
		task := &logTask{
			logPosition: position,
			rootPath:    systemConfig.LogRootPath,
			relatePath:  systemConfig.LogRelatedPath,
			port:        serverConfig.Port,
			stop:        make(chan struct{}, 1),
		}
		tasks.Store(position.Id, task)
		log.Println("注册任务:", position.String())
		return true
	}
	startDay, err := time.Parse("2006-01-02", systemConfig.StartDay)
	if err != nil {
		panic("日志起始日期配置格式(2006-01-02)错误错误" + systemConfig.StartDay)
	}
	id := fmt.Sprintf("%d_%d_%s", serverConfig.Operator, serverConfig.Server, recordName)
	position = &logPosition{
		Id:          id,
		Operator:    serverConfig.Operator,
		Server:      serverConfig.Server,
		Log:         recordName,
		LogType:     logType,
		LastExecute: startDay,
		Position:    0,
		TotalRows:   0,
	}
	// 插入数据库
	SaveToDb(position)

	tasks.Store(position.Id, &logTask{
		logPosition: position,
		rootPath:    systemConfig.LogRootPath,
		relatePath:  systemConfig.LogRelatedPath,
		port:        serverConfig.Port,
		stop:        make(chan struct{}, 1),
	})
	log.Println("注册任务:", position.String())
	return true
}

func StartScanLog(duration time.Duration, process func(recordName string, lines []string)) {
	ticker = time.NewTicker(duration)
	stopChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("开始扫描日志")
				scanAllLog(process)
				log.Println("结束扫描")
			case <-stopChan:
				log.Println("结束扫描任务调度")
				tasks.Range(func(key, value interface{}) bool {
					log.Println(key, "结束扫描")
					return true
				})
				return
			}
		}
	}()
}

func StopScanLog() {
	tasks.Range(func(key, value interface{}) bool {
		logTask := value.(*logTask)
		logTask.Close()
		return true
	})
	if ticker != nil {
		ticker.Stop()
	}
	close(stopChan)
}

func scanAllLog(process func(recordName string, lines []string)) {
	tasks.Range(func(key, value interface{}) bool {
		logTask := value.(*logTask)
		scanOneTask(logTask, process)
		return true
	})
}

func scanOneTask(task *logTask, process func(recordName string, lines []string)) {
	if task.Closed() {
		log.Println(task.logPosition.Id, "任务停止")
		return
	}
	logPosition := task.logPosition
	lastExecute := logPosition.LastExecute
	nowStr := time.Now().Format("2006-01-02")
	now, _ := time.Parse("2006-01-02", nowStr)
	// 处理读取到的行
	var logProcess = func(position int64, lines []string) {
		process(task.logPosition.Log, lines)
		logPosition.Position = position
		logPosition.TotalRows += len(lines)
		SaveToDb(logPosition)
	}
	var stop = func() bool { return task.Closed() }
	// 如果是前一天
	for ; !lastExecute.After(now); lastExecute = lastExecute.Add(24 * time.Hour) {
		if task.Closed() {
			log.Println(task.logPosition.Id, "任务停止")
			return
		}
		sep := string([]byte{os.PathSeparator})
		path := fmt.Sprintf("%s%s%s%s%s%s%s%s%d_%d_%s.%s",
			task.rootPath,
			sep,
			task.port,
			sep,
			task.relatePath,
			sep,
			logPosition.LogType,
			sep,
			logPosition.Operator,
			logPosition.Server,
			logPosition.Log,
			lastExecute.Format("2006-01-02"))

		// 更新扫描的日期
		if lastExecute != logPosition.LastExecute {
			logPosition.LastExecute = lastExecute
			logPosition.Position = 0
			SaveToDb(logPosition)
		}
		// 扫描文件
		scanFile(path, logPosition.Position, logProcess, stop)
	}
}

func scanFile(path string, position int64, process func(position int64, lines []string), stop func() bool) {
	file, err := os.Open(path)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			log.Println(path, "不存在")
			return
		}
		panic(err.Error())
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println("关闭文件句柄错误", path, err.Error())
		}
	}()
	buffer := bytes.NewBuffer(nil)
	cache := make([]byte, 128*1024)
	var offset = position
	ret, err := file.Seek(offset, 0)
	if err != nil {
		log.Panic(path, "Seek失败", err)
		return
	}
	if ret != offset {
		log.Panic(path, offset, ret, "seed位置错误")
	}
	if stop() {
		log.Println(path, "任务停止")
		return
	}
	for {
		read, err := file.Read(cache)
		if read <= 0 {
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err.Error())
			}
			break
		}
		start := 0
		cur := 0
		lines := make([]string, 0, 1)
		for i := 0; i < read; i++ {
			b := cache[i]
			// \n\r
			if b != '\r' && b != '\n' {
				continue
			}
			cur = i

			size := cur - start
			if buffer.Len() > 0 {
				size += buffer.Len()
			}
			if size == 0 {
				// 排除当前字节（因为是换行符）
				start = i + 1
				continue
			}
			if buffer.Len() > 0 {
				buffer.Write(cache[start:cur])
				line := buffer.String()
				lines = append(lines, line)
				buffer.Reset()
			} else {
				line := string(cache[start:cur])
				lines = append(lines, line)
			}
			start = i + 1
		}
		if start < read {
			buffer.Write(cache[start:read])
		}
		// 处理这一批
		process(offset+int64(start), lines)
		offset += int64(read)
		if stop() {
			log.Println(path, "任务停止")
			return
		}
	}
	if buffer.Len() > 0 {
		line := buffer.String()
		log.Println(path, "文件没有结束符[", line, "]")
		process(offset, []string{line})
	}
}
