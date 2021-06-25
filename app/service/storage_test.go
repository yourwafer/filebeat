package service

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"testing"
	"xai.com/shushu/app/model"
)

type TestConfig struct {
	Name     string `storage:"id"`
	Age      int16  `storage:"index,unique"`
	Birthday string
	Model    *Model
	Models   []*Model
	Seg      []int `json:"seg"`
}

func (cfg *TestConfig) String() string {
	return fmt.Sprintf("Id:%s,Age:%d,Birthday:%s,Model:[baseId:%d,Skin:%s],*Models:[-len:%d,baseId:%d,Skin:%s],Seg:%v\r\n",
		cfg.Name, cfg.Age, cfg.Birthday, cfg.Model.BaseId, cfg.Model.Skin, len(cfg.Models), cfg.Models[0].BaseId, cfg.Models[0].Skin, cfg.Seg)
}

type Model struct {
	BaseId int
	Skin   string
}

func TestGetAll(t *testing.T) {
	storage := NewStorage(reflect.TypeOf(TestConfig{}))
	storage.Load("test/Test.xlsx")
	all := storage.GetAll()
	fmt.Println(len(all))
	for _, v := range all {
		fmt.Println(v.(*TestConfig).String())
	}
}

func TestGetById(t *testing.T) {
	storage := NewStorage(reflect.TypeOf(TestConfig{}))
	storage.Load("test/Test.xlsx")
	cfgItem := storage.GetById("wa3").(*TestConfig)
	fmt.Println(cfgItem.String())
}

func TestGetIndex(t *testing.T) {
	storage := NewStorage(reflect.TypeOf(TestConfig{}))
	storage.Load("test/Test.xlsx")
	indexList := storage.GetIndex(int16(2))
	for _, v := range indexList {
		str := v.(*TestConfig).String()
		fmt.Println(str)
	}
}

func TestGetUnique(t *testing.T) {
	storage := NewStorage(reflect.TypeOf(TestConfig{}))
	storage.Load("test/Test.xlsx")
	unique := storage.GetUnique(int16(2)).(*TestConfig)
	fmt.Println(unique.String())
}

func TestMapStructure(t *testing.T) {
	var inter interface{}
	json.Unmarshal([]byte("[1,2,3]"), &inter)
	input := map[string]interface{}{
		"seg": inter,
	}
	testConfig := TestConfig{}
	var obj interface{} = testConfig
	objVal := reflect.ValueOf(&obj)
	fmt.Println(objVal.Kind())
	ele := objVal.Elem()
	fmt.Println(ele.Kind())
	fmt.Println(ele.Elem().Kind())

	err := mapstructure.Decode(input, &obj)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(obj)
}

func TestReflectType(t *testing.T) {
	array := []int{1, 2, 3}
	type1 := reflect.TypeOf(array)
	value1 := reflect.New(type1)
	//value1 := reflect.MakeSlice(type1, 1, 2)
	obj := value1.Interface()
	t1 := reflect.TypeOf(obj)
	fmt.Println(t1.Kind())
	fmt.Println(t1.Elem().Kind())
	fmt.Println(t1.Elem().Elem().Kind())
}

func TestJsonUnDecInterface(t *testing.T) {
	jsonStr := `{"id":"idStr"}`
	var config TestConfig
	var obj interface{} = config
	json.Unmarshal([]byte(jsonStr), &obj)
	fmt.Println(reflect.TypeOf(obj))
}

func TestStruct(t *testing.T) {
	field := model.Field{1, "a"}
	mm := make(map[string]model.Field)
	mm["abc"] = field

	fmt.Printf("%p--%v\r\n", &field, &field)
	m := mm["abc"]
	m.Index = 3
	fmt.Printf("%p--%v\r\n", &m, &m)
}
