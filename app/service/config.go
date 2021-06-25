package service

import (
	"bufio"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"xai.com/shushu/app/model"
)

func LoadAppConfig(path string) *model.AppConfig {
	config := make(map[string]string)

	f, err := os.Open(path)
	defer func() { _ = f.Close() }()
	if err != nil {
		log.Panic(err)
	}

	r := bufio.NewReader(f)
	for {
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		s := strings.TrimSpace(string(b))
		if strings.HasPrefix(s, "#") {
			continue
		}
		index := strings.Index(s, "=")
		if index < 0 {
			continue
		}
		key := strings.TrimSpace(s[:index])
		if len(key) == 0 {
			continue
		}
		value := strings.TrimSpace(s[index+1:])
		if len(value) == 0 {
			continue
		}
		config[key] = value
	}
	appConfig := &model.AppConfig{}
	err = mapstructure.Decode(config, appConfig)
	if err != nil {
		log.Println("读取", path, "解析AppConfig异常", err)
	}
	return appConfig
}

// 加载Excel事件类型配置
func LoadConfig(path string) map[string]*model.EventConfig {
	settingStorage := NewStorage(reflect.TypeOf(model.EventLogSetting{}))
	settingStorage.Load(path)
	commonItems := settingStorage.GetIndex(true)
	commonFields := make(map[string]*model.Field)
	commonSystemFields := make(map[string]*model.Field)
	for _, item := range commonItems {
		setting := item.(*model.EventLogSetting)
		name := setting.Name
		value := &model.Field{Index: setting.CsvIndex, DataType: setting.DataType}
		commonFields[name] = value
		if strings.HasPrefix(name, "#") {
			commonSystemFields[name] = value
		}
	}
	allSetting := settingStorage.GetAll()
	eventConfigs := make(map[string]*model.EventConfig, 10)
	//生成Excel中的所有 EventConfig
	for _, item := range allSetting {
		setting := item.(*model.EventLogSetting)
		if setting.Common {
			continue
		}
		identity := setting.Identity()
		config := eventConfigs[identity]
		if config == nil {
			fields := make(map[string]*model.Field)
			if strings.HasPrefix(setting.SsType, "user_") {
				for k, v := range commonSystemFields {
					fields[k] = v
				}
			} else {
				for k, v := range commonFields {
					fields[k] = v
				}
			}
			config = &model.EventConfig{Fields: fields}
			eventConfigs[identity] = config
		}
		//设置当前属性
		config.PutField(setting.Name, setting.CsvIndex, setting.DataType)
		//日志类型,对应游戏服日志类型如ItemRecord
		config.RecordName = setting.RecordName
		//设置当前事件名
		config.Name = setting.EventName
		//设置日志类型
		config.FileType = setting.LogType
		//设置数数后台处理类型
		config.UploadType = setting.SsType
	}
	return eventConfigs
}

// 加载serverlist配置
func LoadServerConfig(path string) map[string]*model.ServerConfig {
	config := make(map[string]*model.ServerConfig)

	f, err := os.Open(path)
	defer func() { _ = f.Close() }()
	if err != nil {
		log.Panic(err)
	}

	r := bufio.NewReader(f)
	for {
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		s := strings.TrimSpace(string(b))
		if strings.HasPrefix(s, "#") {
			continue
		}
		fields := strings.Fields(s)
		if len(fields) < 3 {
			log.Println(path, s, "配置错误，忽视此行")
			continue
		}
		operator, err := strconv.Atoi(fields[0])
		server, err := strconv.Atoi(fields[1])

		config[fields[0]+fields[1]] = &model.ServerConfig{Operator: operator, Server: server, Port: fields[2]}
	}
	return config
}
