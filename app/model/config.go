package model

type AppConfig struct {
	ExcelPath string
}

type EventSource struct {
	// 来源类型，目前支持flog,tlog,mysql
	sourceType string

	// 如果是flog或者tlog，那么就是日志类型
	name string
}

type Field struct {

	// 数据列下标
	index int

	// 字段类型(number,date,string)
	dataType string
}

// excel 配置内容
type EventConfig struct {
	// 事件名称
	name string

	// 描述信息
	describe string

	// 数据来源
	source EventSource

	// 上报类型 track,user_set,user_del
	uploadType string

	// 字段名称对应数据数组下表
	fields map[string]Field
}
