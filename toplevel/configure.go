package toplevel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var DEFAULT_CONF_FILE = fmt.Sprintf("./%s.conf", SERVERNAME)

var g_conf_file string
var G_Config Configure
var g_conf_etcd string

type SqlCon struct {
	MysqlConn            string
	MysqlConnectPoolSize int
}

type Configure struct {
	MallDb           SqlCon
	RPCListen        string
	LogDir           string
	LogFile          string
	LogRetentionTime int64 //日志保留时间(秒)
}

func GetConfig(config_file string, config *Configure) error {
	fmt.Println("config file:" + config_file)
	file, err := os.Open(config_file)
	if err != nil {
		return err
	}
	defer file.Close()

	config_str, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(config_str, config)
	if err != nil {
		fmt.Println("parase config error for:" + err.Error())
	}
	return err
}

func InitConfig() {
	var err error
	if g_conf_file != "" {
		goto LoadConfig
	} else {
		fmt.Fprintf(os.Stderr, "No configuration source\n")
		os.Exit(1)
	}

LoadConfig:
	//init config
	err = GetConfig(g_conf_file, &G_Config)
	if err != nil {
		log.Fatal("parse config file error: %s", err.Error())
		return
	}
	fmt.Printf("Config:%+v\n", G_Config)
	return
}
