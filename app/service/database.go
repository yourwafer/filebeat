package service

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

type logPosition struct {
	Id string
	//运营商
	Operator int

	// 服务器
	Server int

	// 日志名称
	Log string

	// 日志类型tlog，flog
	LogType string

	// 上次处理时间
	LastExecute time.Time

	// 上次读取位置
	Position int64

	// 累计行数
	TotalRows int
}

func (p *logPosition) String() string {
	return fmt.Sprintf("%s,%s,%s,pos=%d,total=%d", p.Id, p.LogType, p.LastExecute.String(), p.Position, p.TotalRows)
}

var dbPool *sql.DB
var once sync.Once

func InitDatabase(userName, password, addr, database string) {
	once.Do(func() {
		initDatabase(userName, password, addr, database)
	})
}

func initDatabase(userName string, password string, addr string, database string) {
	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s?charset=utf8&parseTime=True", userName, password, "tcp", addr, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("mysql初始化失败" + dsn + ";" + err.Error())
	}
	dbPool = db
	dbPool.SetConnMaxLifetime(60 * time.Second) //最大连接周期，超过时间的连接就close
	dbPool.SetMaxOpenConns(5)                   //设置最大连接数
	dbPool.SetMaxIdleConns(2)                   //设置闲置连接数

	content, err := ioutil.ReadFile("config/init.sql")
	result, err := dbPool.Exec(string(content))
	if err != nil {
		panic("初始化LogPosition表失败" + err.Error())
	}
	affect, err := result.RowsAffected()
	if err != nil {
		panic("插入数据库失败" + err.Error())
	}
	if affect > 0 {
		log.Println("初始化数据库成功,新建数据表")
	} else {
		log.Println("初始化数据库成功")
	}
}

func LoadOne(operator, server int, recordName string) *logPosition {
	if dbPool == nil {
		panic("数据库未完成初始化")
	}
	id := fmt.Sprintf("%d_%d_%s", operator, server, recordName)
	stmt, err := dbPool.Prepare("select `id`,`operator`,`server`,`log`,`type`,`last_execute`,`position`,`total_rows` from log_position where id=?")
	defer func() {
		err := stmt.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		panic("数据库查询失败" + err.Error())
	}
	row := stmt.QueryRow(id)
	entity := &logPosition{}
	err = row.Scan(&entity.Id, &entity.Operator, &entity.Server, &entity.Log, &entity.LogType, &entity.LastExecute, &entity.Position, &entity.TotalRows)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		panic("数据库查询失败" + err.Error())
	}
	return entity
}

func SaveToDb(position *logPosition) {
	if dbPool == nil {
		panic("数据库未完成初始化")
	}
	preEntity := LoadOne(position.Operator, position.Server, position.Log)
	var result sql.Result
	var err error
	if preEntity != nil {
		stmt, preErr := dbPool.Prepare("update log_position set `last_execute`=?,`position`=?,`total_rows`=? where `id`=?")
		if preErr != nil {
			panic("数据库更新失败" + preErr.Error())
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				log.Println(err)
			}
		}()
		result, err = stmt.Exec(position.LastExecute, position.Position, position.TotalRows, position.Id)
	} else {
		stmt, preErr := dbPool.Prepare("insert into log_position(`id`,`operator`,`server`,`log`,`type`,`last_execute`,`position`,`total_rows`) values(?,?,?,?,?,?,?,?)")
		if preErr != nil {
			panic("数据库插入失败" + preErr.Error())
		}
		defer func() {
			err := stmt.Close()
			if err != nil {
				log.Println(err)
			}
		}()
		result, err = stmt.Exec(position.Id, position.Operator, position.Server, position.Log, position.LogType, position.LastExecute, position.Position, position.TotalRows)
	}

	if err != nil {
		panic(position.Id + "保存记录操作失败" + err.Error())
	}
	_, err = result.RowsAffected()
	if err != nil {
		panic(position.Id + "保存记录Row失败" + err.Error())
	}
}
