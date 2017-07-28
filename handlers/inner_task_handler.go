package handlers

import (
	"github.com/astaxie/beego/orm"
	bt "github.com/lanfang/go-crontab/basetype"
	cron "github.com/lanfang/go-crontab/crontab"
	md "github.com/lanfang/go-crontab/models"
	tp "github.com/lanfang/go-crontab/toplevel"
	"github.com/lanfang/go-lib/log"
	"time"
)

func ClearLogHandler(tr interface{}) error {
	arg, ok := tr.(*bt.CallBackArg)
	log.Info("Begin ClearLogHandler %+v", arg.NodeInfo)
	if !ok {
		log.Info("End ClearLogHandler 参数错误 %+v", arg.NodeInfo)
		return bt.ErrInvalidArgument
	}
	o := orm.NewOrm()
	limit_szie := 30000
	end_time := time.Now().Add(-1 * (time.Second * time.Duration(tp.G_Config.LogRetentionTime)))
	for loop := true; loop; {
		res, err := o.Raw("DELETE from task_log where begin_time <= ? limit ?", end_time, limit_szie).Exec()
		cnt, err2 := res.RowsAffected()
		log.Info("End ClearLogHandler cnt:%+v, err2:%+v, err:%+v, arg.NodeInfo:%+v", cnt, err2, err, arg.NodeInfo)
		if cnt == 0 {
			loop = false
		}
	}
	cron.Crontab.AddRepeatTask(arg.NodeInfo)
	return nil
}

func InitInnerCron() {
	//清理日志定时任务
	clear_log_time := "0 0 */4 * * *"
	clear_log_task := md.Task{Type: bt.InnerTypeClearLog, StartTime: clear_log_time}
	cron.Crontab.AddTask(clear_log_task, bt.Normal, ClearLogHandler)
}
