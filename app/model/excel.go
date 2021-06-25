package model

/**
excel 配置类
*/
type EventLogSetting struct {

	// 无意义id
	Id int

	//上传数据的标识名
	Name string

	//对应的日志下标
	CsvIndex byte

	//对应的java类型
	DataType string

	//对应的数数处理的type类型
	SsType string

	//事件名
	EventName string

	//后台日志名
	RecordName string

	// 公共字段
	Common bool `storage:"index"`

	//日志类型
	LogType string
}

func (that *EventLogSetting) Identity() string {
	return that.RecordName + "_" + that.SsType + "_" + that.EventName
}
