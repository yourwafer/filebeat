package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"xai.com/shushu/app/model"
	"xai.com/shushu/app/service"
)

func main() {
	// 系统配置
	appConfig := service.LoadAppConfig("config/application.properties")
	// 开启线上状态监控
	if len(appConfig.StartPprof) > 0 {
		go func() { _ = http.ListenAndServe(appConfig.StartPprof, nil) }()
	}
	// 日志类型配置
	eventConfigs := service.LoadConfig(appConfig.ExcelPath)
	// 按照日志名分类
	eventConfigByRecordName := classifyByRecordName(eventConfigs)
	// 服务器列表
	serverConfigs := service.LoadServerConfig(appConfig.ServerList)

	// 初始化数据库
	service.InitDatabase(appConfig.MysqlUser, appConfig.MysqlPassword, appConfig.MysqlAddr, appConfig.MysqlDatabase)

	// 注册扫描任务
	registerEvent(serverConfigs, eventConfigByRecordName, appConfig)
	// 初始化处理每行内容处理器
	service.InitConsumer(appConfig)
	log.Println("开始扫描任务")
	interval, _ := strconv.Atoi(appConfig.LogProcessInterval)
	processInterval := time.Duration(interval) * time.Second

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	// 开始扫描任务
	service.StartScanLog(processInterval, func(recordName string, lines []string) {
		eventConfigs := eventConfigByRecordName[recordName]
		service.Process(recordName, lines, eventConfigs)
	})
	// 定时读取serverlist文件，运维会动态修改此文件
	if len(appConfig.ServerListReRead) > 0 {
		interval, _ := strconv.ParseInt(appConfig.ServerListReRead, 10, 64)
		if interval > 0 {
			if interval < 60 {
				interval = 60
			}
			go func() {
				ticker := time.NewTicker(time.Duration(interval) * time.Second)
				for range ticker.C {
					serverConfigs := service.LoadServerConfig(appConfig.ServerList)
					registerEvent(serverConfigs, eventConfigByRecordName, appConfig)
				}
			}()
		}
	}
	sig := <-signals
	log.Println("收到信号,准备关闭所有任务", sig.String())
	service.StopScanLog()
	time.Sleep(2 * time.Second)
	log.Println("进程已经正确停止")
}

func registerEvent(serverConfigs map[string]*model.ServerConfig, eventConfigByRecordName map[string][]*model.EventConfig, appConfig *model.AppConfig) {
	for _, serverConfig := range serverConfigs {
		for _, eventConfig := range eventConfigByRecordName {
			if len(eventConfig) == 0 {
				continue
			}
			oneOf := eventConfig[0]
			service.RegisterEvent(appConfig, serverConfig, oneOf.RecordName, oneOf.FileType)
		}
	}
}

func classifyByRecordName(configs map[string]*model.EventConfig) map[string][]*model.EventConfig {
	result := make(map[string][]*model.EventConfig)
	for _, v := range configs {
		pre := result[v.RecordName]
		if pre == nil {
			pre = make([]*model.EventConfig, 0, 1)
		}
		pre = append(pre, v)
		result[v.RecordName] = pre
	}
	return result
}
