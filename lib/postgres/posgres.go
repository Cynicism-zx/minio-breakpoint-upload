package postgres

import (
	"fmt"

	"oss/config"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type global struct {
	DB *gorm.DB
}

var Global global

func Init() {
	dbDriver := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.PostgresUsername, config.PostgresPassword, config.PostgresDbName, config.PostgresPort)
	db, err := gorm.Open("postgres", dbDriver)
	if err != nil {
		fmt.Println("open db failed:" + err.Error())
		panic(err)
	}

	//defer db.Close()

	db.SingularTable(true)
	db.LogMode(true)

	// SetMaxIdleCons 设置连接池中的最大闲置连接数。
	db.DB().SetMaxIdleConns(10)

	// SetMaxOpenCons 设置数据库的最大连接数量。
	db.DB().SetMaxOpenConns(100)

	//db, err := pg.NewDB(&pg.DBConfig{
	//	DbType:      "postgres",
	//	Host:        config.PostgresIp,
	//	Port:        config.PostgresPort,
	//	User:        config.PostgresUsername,
	//	Password:    config.PostgresPassword,
	//	DbName:      config.PostgresDbName,
	//	SslMode:     "disable",
	//	NotPrintSql: false,
	//	LogLevel:    "debug",
	//})
	//if err != nil {
	//	logger.LOG.Error("open db failed:" + err.Error())
	//	return
	//}

	Global.DB = db
	return
}
