package service

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xai.com/shushu/app/model"
)

var (
	processor        []func(string, []string, []*model.EventConfig)
	httpClient       *http.Client
	uploadUrl        string
	uploadAppId      string
	ignoreFieldError bool
)

func InitConsumer(config *model.AppConfig) {
	ignoreFieldError, _ = strconv.ParseBool(config.IgnoreFieldError)
	processor = make([]func(string, []string, []*model.EventConfig), 0, 2)
	if strings.Contains(config.PushType, "console") {
		processor = append(processor, consoleProcess)
	}
	if strings.Contains(config.PushType, "http") {
		processor = append(processor, httpProcess)
		httpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 10,
				MaxConnsPerHost:     10,
				IdleConnTimeout:     time.Minute,
			},
			Timeout: time.Duration(3) * time.Second,
		}
		uploadUrl = config.HttpServerUrl
		uploadAppId = config.HttpAppId
	}
}

func Process(recordName string, lines []string, eventConfigs []*model.EventConfig) {
	for _, process := range processor {
		process(recordName, lines, eventConfigs)
	}
}

// 标准输出
func consoleProcess(recordName string, lines []string, eventConfigs []*model.EventConfig) {
	for _, line := range lines {
		fmt.Println(recordName, line)
	}
}

// HTTP上报
func httpProcess(recordName string, lines []string, eventConfigs []*model.EventConfig) {
	if lines == nil {
		return
	}
	linesSize := len(lines)
	if linesSize == 0 {
		return
	}
	lineSplits := splitLines(lines)
	for _, eventConfig := range eventConfigs {
		rows := make([]map[string]interface{}, 0, linesSize)
		for _, cols := range lineSplits {
			var values = parse(eventConfig, cols)
			processDefaultProperties(eventConfig, values)

			dateTime := values["#time"]
			if dateTime == nil {
				jsonStr, _ := json.Marshal(values)
				log.Println("解析[{}]出现异常时间空置数据[{}]", eventConfig.Name, jsonStr)
				continue
			}

			rows = append(rows, values)
		}
		log.Println("解析类型", eventConfig.RecordName, eventConfig.UploadType, "数据行数", len(rows))
		jsonValue, err := json.Marshal(rows)
		if err != nil {
			panic(err.Error())
		}
		retryHttpPost(jsonValue, 1)
	}
}

func retryHttpPost(value []byte, retry int) {
	defer func() {
		if e := recover(); e != nil {
			if retry >= 3 {
				log.Panic("重试次数达到", retry, "进程直接退出")
			}
			sleepWait := time.Second
			log.Println("http发送失败，等待重试", sleepWait)
			time.Sleep(sleepWait)
			retry++
			retryHttpPost(value, retry)
		}
	}()
	//httpPost(value)
}

func httpPost(jsonValue []byte) {
	resultBuffer := bytes.NewBuffer(nil)
	writer := gzip.NewWriter(resultBuffer)
	_, err := writer.Write(jsonValue)
	if err != nil {
		if writer != nil {
			_ = writer.Close()
		}
		log.Panic("压缩参数内容失败", len(jsonValue), err)
	}
	_ = writer.Close()
	afterGzip := resultBuffer.Len()
	request, err := http.NewRequest("POST", uploadUrl, resultBuffer)
	if err != nil {
		log.Panic("构建参数异常", len(jsonValue), afterGzip, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("appid", uploadAppId)
	request.Header.Set("compress", "gzip")
	startTime := time.Now()
	res, err := httpClient.Do(request)
	duration := time.Now().Sub(startTime)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}
	if err != nil {
		log.Panic("上报数据失败", len(jsonValue), err.Error())
	}
	if res.StatusCode != 200 {
		log.Panic("上报数据失败", len(jsonValue), res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic("读取上报返回异常", len(jsonValue), err.Error())
	}
	shuShuRes := &model.ShuShuHttpRes{}
	err = json.Unmarshal(body, shuShuRes)
	if err != nil {
		log.Panic("解析上报返回失败", err.Error())
	}
	if shuShuRes.Code == 0 {
		log.Println("上报数数数据成功", afterGzip, duration)
		return
	}
	switch shuShuRes.Code {
	case -1:
		log.Panic("数数上报异常", shuShuRes.Msg, "invalid data format")
	case -2:
		log.Panic("数数上报异常", shuShuRes.Msg, "APP ID doesn't exist")
	case -3:
		log.Panic("数数上报异常", shuShuRes.Msg, "invalid ip transmission")
	default:
		log.Panic("Unexpected response return code", shuShuRes.Code)
	}
}

func processDefaultProperties(eventConfig *model.EventConfig, values map[string]interface{}) {
	uploadType := eventConfig.UploadType
	values["#type"] = uploadType
	if "track" == uploadType {
		eventName := eventConfig.Name
		values["#event_name"] = eventName
	} else if ("user_set" != uploadType) && ("user_setOnce" != uploadType) {
		return
	}

	account := values["#account_id"].(string)
	if account == "" || "-1" == account {
		return
	}

	index := strings.LastIndex(account, ".")
	if index >= 0 {
		account = string([]byte(account)[0:index])
	}
	indexOf := strings.LastIndex(account, "_")
	var userId string
	if indexOf <= 0 {
		userId = account
	} else {
		userId = string([]byte(account)[0:indexOf])
	}
	//noinspection unchecked
	properties := values["properties"]
	if properties == nil {
		properties = map[string]interface{}{"userId": userId}
		values["properties"] = properties
	} else {
		properties.(map[string]interface{})["userId"] = userId
	}
}

func parse(eventConfig *model.EventConfig, cols []string) map[string]interface{} {
	fields := eventConfig.Fields
	values := make(map[string]interface{}, len(fields))

	values["#type"] = eventConfig.UploadType

	properties := make(map[string]interface{})
	for name, field := range fields {
		index := field.Index
		if int(index) > len(cols) {
			log.Println("日志[{}]列[{}]下标[{}]大约最大值[{}]", eventConfig.Name, name, index, len(cols))
			continue
		}
		fieldType := field.DataType
		strValue := cols[index-1]
		if "server" == name {
			strValue = "9999"
		}
		var value interface{}
		var err error
		switch fieldType {
		case "string":
			if len(strValue) > 1024 {
				strValue = string(([]byte(strValue))[0:1024])
			}
			value = strValue
		case "int":
			value, err = strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				if ignoreFieldError {
					value = 0
				} else {
					log.Panic("解析", name, "失败", strValue, err.Error())
				}
			}
		case "float":
			value, err = strconv.ParseFloat(strValue, 64)
			if err != nil {
				if ignoreFieldError {
					value = 0.0
				} else {
					log.Panic("解析", name, "失败", strValue, err.Error())
				}
			}
		case "date":
			millSec, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				if ignoreFieldError {
					millSec = 0
				} else {
					log.Panic("解析", name, "失败", strValue, err.Error())
				}
			}
			if "#time" == name && err != nil {
				joinStr := strings.Join(cols, "@")
				log.Println("解析日期字段错误，忽视数据行>>", joinStr)
				return nil
			}
			curTime := time.Unix(0, int64(time.Duration(millSec)*time.Millisecond))
			value = curTime.Format("2006-01-02 15:04:05.000")
			if curTime.After(time.Now()) {
				joinStr := strings.Join(cols, "@")
				log.Println(eventConfig.Name, "解析出日期大于当前日期,忽视当前行", curTime, strValue, ">>", joinStr)
				continue
			}
		case "bool":
			value, _ = strconv.ParseBool(strValue)
		case "[I":
			array := make([]int, 0, 3)
			err = json.Unmarshal([]byte(strValue), &array)
			if err != nil && !ignoreFieldError {
				log.Panic("解析", name, "失败", strValue, err.Error())
			}
			value = array
		}

		if strings.HasPrefix(name, "#") {
			values[name] = value
		} else if strings.HasPrefix(name, "${") {
			properties[getRealName(name, cols)] = value
		} else {
			properties[name] = value
		}
	}

	if len(properties) > 0 {
		values["properties"] = properties
	}

	return values
}

func getRealName(name string, cols []string) string {
	nameByte := []byte(name)
	index := nameByte[strings.Index(name, "{")+1 : strings.Index(name, "}")]
	indexValue, err := strconv.Atoi(string(index))
	if err != nil {
		panic(err.Error())
	}
	return cols[indexValue-1]
}

func splitLines(lines []string) [][]string {
	fieldLines := make([][]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		fieldLines = append(fieldLines, fields)
	}
	return fieldLines
}
