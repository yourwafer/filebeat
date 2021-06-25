package model

type AppConfig struct {
	ExcelPath          string // Excel事件配置文件路径
	ServerList         string // 运维serverlist配置路径
	LogRootPath        string // 游戏服务器日志根路径
	LogRelatedPath     string // 日志相对路径(根路径/port/相对路径/[tlog|flog]/1_1_LogType.yyyy-MM-dd)
	LogProcessInterval string //日志重新读取间隔
	PushType           string //日志输出类型,console:控制台输出,http:上报数数平台
	HttpServerUrl      string //http数数上报url
	HttpAppId          string //http数数上报appid
	StartDay           string // 开始上报日志的时间,格式2021-04-20
	MysqlUser          string //mysql账号
	MysqlPassword      string //mysql密码
	MysqlDatabase      string //mysql数据库
	MysqlAddr          string // mysql地址
	StartPprof         string //开启线上监控
	IgnoreFieldError   string // 忽视字段解析失败
	ServerListReRead   string // 循环间隔读取serverlist文件
}

type EventSource struct {
	// 来源类型，目前支持flog,tlog
	SourceType string

	// 如果是flog或者tlog，那么就是日志类型
	Name string
}

type Field struct {

	// 数据列下标
	Index byte

	// 字段类型(int,float,date,string,bool,[I)
	DataType string
}

// excel 配置内容
type EventConfig struct {
	// 事件名称,对应数数事件名称
	Name string

	// 日志类型,对应游戏服日志类型如ItemRecord
	RecordName string

	// 数据来源
	FileType string

	// 上报类型 track,user_set,user_del
	UploadType string

	// 字段名称对应数据数组下表
	Fields map[string]*Field
}

func (that *EventConfig) PutField(name string, index byte, dataType string) {
	that.Fields[name] = &Field{Index: index, DataType: dataType}
}

type ServerConfig struct {
	// 运营商
	Operator int

	// 服务器
	Server int

	// 端口
	Port string
}
