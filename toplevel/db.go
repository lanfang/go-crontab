package toplevel

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
)

var G_ORM orm.Ormer

func InitBeegoOrm() {
	orm.RegisterDriver("mysql", orm.DRMySQL)
	orm.RegisterDataBase("default", "mysql", G_Config.MallDb.MysqlConn, G_Config.MallDb.MysqlConnectPoolSize/2,
		G_Config.MallDb.MysqlConnectPoolSize)
	orm.Debug = true
	sql_log_fp, err := os.OpenFile(G_Config.LogDir+"/"+G_Config.LogFile+".mysql", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open file[%s.mysql] failed[%s]", G_Config.LogFile, err)
		return
	}

	orm.DebugLog = orm.NewLog(sql_log_fp)

	o := orm.NewOrm()
	o.Using("default")

	G_ORM = o
}
