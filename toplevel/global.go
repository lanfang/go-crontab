package toplevel

import (
	"flag"
	"fmt"
	"github.com/lanfang/go-lib/log"
	"os"
)

var SERVERNAME string = "go-crontab"

func InitGlobal() {
	const usage = "go-crontab [-c config_file][-p cpupro file][-m mempro file] [-etcd etcd_addr]"
	flag.StringVar(&g_conf_file, "c", "", usage)
	flag.StringVar(&g_cpupro_file, "p", "", usage)
	flag.StringVar(&g_mempro_file, "m", "", usage)
	flag.StringVar(&g_conf_etcd, "etcd", "", usage)
	flag.Parse()
	err := InitCPUProfile()
	if err != nil {
		fmt.Println("error init CUPProfile %s" + err.Error())
		os.Exit(1)
	}
	InitConfig()
	InitBeegoOrm()
	log.Gen(SERVERNAME, G_Config.LogDir, G_Config.LogFile)
}
