package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/json-iterator/go"
)

var PostgresIp string
var PostgresUsername string
var PostgresPassword string
var PostgresPort string
var PostgresDbName string
var PORT string
var MinioAddress string
var MinioAccessKeyId string
var MinioSecretAccessKey string
var MinioSecure string
var MinioBucket string
var MinioBasePath string
var MinioLocation string

func loadFromConfigFile(configFilePath string) error {
	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println("open config file failed:" + err.Error())
		panic(err)
	}

	data, err := ioutil.ReadAll(file)

	if nil != err {
		return err
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var jsonConfig jsoniter.Any = json.Get(data)

	PostgresIp = jsonConfig.Get("POSTGRES_IP").ToString()
	PostgresUsername = jsonConfig.Get("POSTGRES_USERNAME").ToString()
	PostgresPassword = jsonConfig.Get("POSTGRES_PASSWORD").ToString()
	PostgresDbName = jsonConfig.Get("POSTGRES_DBNAME").ToString()
	PostgresPort = jsonConfig.Get("POSTGRES_PORT").ToString()
	PORT = jsonConfig.Get("PORT").ToString()
	MinioAddress = jsonConfig.Get("MINIO_ADDRESS").ToString()
	MinioAccessKeyId = jsonConfig.Get("MINIO_ACCESS_KEY_ID").ToString()
	//keyTmp := jsonConfig.Get("MINIO_SECRET_ACCESS_KEY").ToString()
	MinioSecretAccessKey = jsonConfig.Get("MINIO_SECRET_ACCESS_KEY").ToString()
	MinioSecure = jsonConfig.Get("MINIO_SECURE").ToString()
	MinioBucket = jsonConfig.Get("MINIO_BUCKET").ToString()
	MinioBasePath = jsonConfig.Get("MINIO_BASE_PATH").ToString()
	MinioLocation = jsonConfig.Get("MINIO_LOCATION").ToString()

	if PostgresIp == "" || PostgresUsername == "" || PostgresPassword == "" || PostgresPort == "" ||
		PORT == "" || PostgresDbName == "" || MinioAddress == "" || MinioAccessKeyId == "" ||
		MinioSecretAccessKey == "" || MinioSecure == "" {
		return errors.New("config is error")
	}

	return nil
}

func Init() {
	configFile := "config.json"
	err := loadFromConfigFile(configFile)
	if nil != err {
		fmt.Println("load config file failed:" + err.Error())
		return
	}

	return
}
