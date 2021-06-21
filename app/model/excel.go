package model

/**
excel 配置类
*/
type EventLogSetting struct {

	// 无意义id
	id int

	//上传数据的标识名
	name string

	//对应的日志下标
	csvIndex byte

	//对应的java类型
	dataType byte

	//对应的数数处理的type类型
	ssType string

	//事件名
	eventName string

	//后台日志名
	recordName string

	common bool

	//日志类型
	logType string
}
