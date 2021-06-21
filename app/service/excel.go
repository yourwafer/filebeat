package service

import (
	"encoding/json"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/mitchellh/mapstructure"
	"log"
	"reflect"
	"strconv"
	"strings"
)

type Storage struct {
	ValueType reflect.Type                  // 类型
	values    map[interface{}]interface{}   // 根据值ID数据集合
	indexes   map[interface{}][]interface{} // 自定义索引集合
	unique    map[interface{}]interface{}   // 唯一索引
}

type Indexer interface {
	Index() string
}

type Uniquer interface {
	Unique() string
}

func New(valueType reflect.Type) *Storage {
	return &Storage{ValueType: valueType}
}

type fieldInfo struct {
	index int
	name  *string
}

func (st *Storage) Load(path string) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		log.Panicln("Excel配置路径不存在[", path, "]")
	}
	sheetMap := f.GetSheetMap()
	fieldList := make([]fieldInfo, 0)
	const (
		ID          = "id"
		UNIQUE      = "unique"
		INDEX       = "index"
		ID_FLAG     = 1 << 0
		UNIQUE_FLAG = 1 << 1
		INDEX_FLAG  = 1 << 2
	)
	numField := st.ValueType.NumField()
	fieldFlags := make(map[string]int, 1)
	gotId := false
	for i := 0; i < numField; i++ {
		struField := st.ValueType.Field(i)
		tags := struField.Tag.Get("storage")
		flag := 0
		if "Id" == struField.Name || strings.Contains(tags, ID) {
			flag |= ID_FLAG
		}
		if strings.Contains(tags, INDEX) {
			flag |= INDEX_FLAG
		}
		if strings.Contains(tags, UNIQUE) {
			flag |= UNIQUE_FLAG
		}
		if (flag & ID_FLAG) == ID_FLAG {
			gotId = true
		}
		fieldFlags[struField.Name] = flag
	}
	if !gotId {
		panic("表格" + path + "没有ID配置")
	}
	st.values = make(map[interface{}]interface{}, 16)
	st.indexes = make(map[interface{}][]interface{}, 1)
	st.unique = make(map[interface{}]interface{})
SHEET_LOOP:
	for _, sheetName := range sheetMap {
		rows := f.GetRows(sheetName)
		log.Println("开始解析Excel", path, ",分页:", sheetName, ",rows:", strconv.FormatInt(int64(len(rows)), 10))
		if len(rows) == 0 {
			continue
		}
		// 如果每一页第一个cell为空，说明此Excel被忽略
		if len(rows[0]) == 0 {
			continue
		}
		firstCell := rows[0][0]
		if len(firstCell) == 0 {
			continue
		}
		dataStar := false
		fieldList = fieldList[:0]
		for rowIndex, row := range rows {
			firstCell = row[0]

			if "END_BEFORE" == firstCell {
				continue SHEET_LOOP
			}
			rowLen := len(row)
			// 还没找到表头
			if !dataStar {
				if "SERVER" != firstCell {
					continue
				}
				for i := 1; i < rowLen; i++ {
					curCell := row[i]
					if len(curCell) == 0 {
						continue
					}
					curCell = strings.TrimSpace(curCell)
					if len(curCell) == 0 {
						continue
					}
					fieldList = append(fieldList, fieldInfo{index: i, name: &curCell})
				}
				if len(fieldList) == 0 {
					continue SHEET_LOOP
				}
				dataStar = true
				continue
			}
			instance := reflect.New(st.ValueType)
			findId := false
			for _, field := range fieldList {
				fieldIndex := field.index
				// 存在空列
				if fieldIndex >= rowLen {
					continue
				}
				canalFieldName := canalName(field.name)
				fieldValue := instance.Elem().FieldByName(canalFieldName)
				if !fieldValue.CanAddr() {
					continue
				}
				curCell := row[fieldIndex]
				if len(curCell) == 0 {
					switch fieldValue.Kind() {
					case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
						fieldValue.Set(reflect.Zero(fieldValue.Type()))
					}
					continue
				}
				switch fieldValue.Type().Kind() {
				case reflect.String:
					fieldValue.SetString(curCell)
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					n, err := strconv.ParseInt(curCell, 10, 64)
					if err != nil || fieldValue.OverflowInt(n) {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell)
					}
					fieldValue.SetInt(n)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					n, err := strconv.ParseUint(curCell, 10, 64)
					if err != nil || fieldValue.OverflowUint(n) {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell)
					}
					fieldValue.SetUint(n)
				case reflect.Float32, reflect.Float64:
					n, err := strconv.ParseFloat(curCell, fieldValue.Type().Bits())
					if err != nil || fieldValue.OverflowFloat(n) {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell)
					}
					fieldValue.SetFloat(n)
				case reflect.Bool:
					if len(curCell) != 0 {
						if "true" == curCell || "TRUE" == curCell {
							fieldValue.SetBool(true)
						}
					}
				case reflect.Struct:
					panic(st.ValueType.Kind().String() + ".canalFieldName不能为结构体，请使用结构体指针*Struct")
				case reflect.Ptr:
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
					err := json.Unmarshal([]byte(curCell), fieldValue.Interface())
					if err != nil {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell + "__" + err.Error())
					}
				case reflect.Slice:
					slicePtr := reflect.New(fieldValue.Type()).Elem().Interface()
					var obj interface{}
					err := json.Unmarshal([]byte(curCell), &obj)
					if err != nil {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell + "__" + err.Error())
					}
					err = mapstructure.Decode(obj, &slicePtr)
					if err != nil {
						panic("表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell + "__" + err.Error())
					}
					fieldValue.Set(reflect.ValueOf(slicePtr))
				}

				// id字段
				flag := fieldFlags[canalFieldName]
				fieldInstance := fieldValue.Interface()
				rowData := instance.Interface()
				if (flag & ID_FLAG) == ID_FLAG {
					findId = true
					st.values[fieldInstance] = rowData
				}
				if (flag & INDEX_FLAG) == INDEX_FLAG {
					list := st.indexes[fieldInstance]
					if list == nil {
						list = make([]interface{}, 0, 1)
					}
					list = append(list, rowData)
					st.indexes[fieldInstance] = list
				}
				if (flag & UNIQUE_FLAG) == UNIQUE_FLAG {
					pre := st.unique[fieldInstance]
					if pre != nil {
						panic("表格唯一索引冲突" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + curCell)
					}
					st.unique[fieldInstance] = rowData
				}
			}
			if !findId {
				curRow, _ := json.Marshal(row)
				panic("没有找到ID,表格" + path + "分页" + sheetName + "行" + strconv.FormatInt(int64(rowIndex), 10) + "<>" + string(curRow))
			}
			if "END" == firstCell {
				continue SHEET_LOOP
			}
		}
	}
}

func canalName(name *string) string {
	bb := []byte(*name)
	if bb[0] < 'a' {
		return *name
	}
	bb[0] = bb[0] - ('a' - 'A')
	return string(bb)
}

func (st *Storage) GetAll() []interface{} {
	cache := make([]interface{}, 0, len(st.values))
	for _, v := range st.values {
		cache = append(cache, v)
	}
	return cache
}

func (st *Storage) GetById(id interface{}) interface{} {
	return st.values[id]
}

func (st *Storage) GetIndex(index interface{}) []interface{} {
	return st.indexes[index]
}

func (st *Storage) GetUnique(i interface{}) interface{} {
	return st.unique[i]
}
