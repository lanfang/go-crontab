package models

import (
	"github.com/astaxie/beego/orm"
	bt "github.com/lanfang/go-crontab/basetype"
	"time"
)

func init() {
	orm.RegisterModel(new(Task), new(TaskLog))
}

//任务配置
type Task struct {
	Id         int64          `orm:"pk" json:"task_id"` //任务ID
	Name       string         `json:"name"`             //任务描述
	Enable     bt.EnableType  `json:"enable"`           //是否开启任务, 1 开启,2 停用
	Module     int            `json:"module"`           //该任务隶属模块
	Type       bt.NodeType    `json:"type"`             //任务类型
	StartTime  string         `json:"start_time"`       //任务开始时间,可以是绝对时间和通配时间,绝对时间格式: 时间戳; 通配时间格式:秒,分,时,日,月,星期
	Url        string         `json:"url"`              //回调的url
	Method     string         `json:"method"`           //请求url的方法
	RespRegexp string         `json:"resp_regexp"`      //url回包正则，调用时用于判断是否为预期回包
	Body       string         `json:"body"`             //body字符串
	BodyType   bt.ContentType `json:"body_type"`        //Content-Type: 1 Json, 2 Form
	RedoParam  string         `json:"redo_param"`       //重试间隔, 用空格分割,单位为秒,比如:2 4 8
	Status     bt.StatusType  `json:"status"`           //执行状态, 1 开始执行，2 执行成成功，3 执行失败
	BeginTime  time.Time      `json:"begin_time"`       //开始时间
}

func (p *Task) TableName() string {
	return "task_config_info"
}

func (p *Task) FindTaskByTaskId(o orm.Ormer, task_id int64) error {
	return o.QueryTable(p).Filter("id", task_id).One(p)
}

func (p *Task) FindTaskByName(o orm.Ormer, name string) error {
	return o.QueryTable(p).Filter("name", name).One(p)
}

func (p *Task) Insert(o orm.Ormer) (int64, error) {
	id, err := o.Insert(p)
	return id, err
}

func (p *Task) UpdateByTaskId(o orm.Ormer, task_id int64, fields orm.Params) error {
	_, err := o.QueryTable(p).Filter("id", task_id).Update(fields)
	return err
}

func (p *Task) DeleteByTaskId(o orm.Ormer, task_id int64) error {
	_, err := o.QueryTable(p).Filter("id", task_id).Delete()
	return err
}

func (p *Task) UpdateByName(o orm.Ormer, name string, fields orm.Params) error {
	_, err := o.QueryTable(p).Filter("name", name).Update(fields)
	return err
}

func (p *Task) FindEnableTask(o orm.Ormer) ([]Task, error) {
	var rs []Task
	_, err := o.QueryTable(p).Filter("enable", bt.Enable).Limit(-1).All(&rs)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

//任务日志
type TaskLog struct {
	Id          int64         `orm:"pk"`
	TaskId      int64         `json:"task_id"` //任务ID 自动生成
	Name        string        `json:"name"`    //任务描述
	Module      int           //该任务隶属模块
	RunningType bt.RunType    `json:"run_type"`   //任务执行方式 1 正常启动，2 重试
	Status      bt.StatusType `json:"status"`     //执行状态, 1 开始执行，2 执行成成功，3 执行失败
	BeginTime   time.Time     `json:"begin_time"` //开始时间
	EndTime     time.Time     `json:"end_time"`   //结束时间
	Url         string        `json:"url"`        //回调的url
	Method      string        `json:"method"`     //请求url的方法
	Body        string        `json:"body"`       //body字符串
	Resp        string        `json:"resp"`       //url 的回包内容

}

func (p *TaskLog) TableName() string {
	return "task_log"
}

func (p *TaskLog) FindLogByTaskId(o orm.Ormer, task_id int64) ([]TaskLog, error) {
	var rs []TaskLog
	_, err := o.QueryTable(p).Filter("task_id", task_id).Limit(-1).All(&rs)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func (p *TaskLog) Insert(o orm.Ormer) error {
	log_id, err := o.Insert(p)
	p.Id = log_id
	return err
}

func (p *TaskLog) FindByTaskIdOrName(o orm.Ormer, id_list []int64, name_list []string) ([]TaskLog, error) {
	var rs []TaskLog
	cond := orm.NewCondition()
	if len(id_list) > 0 {
		cond = cond.AndCond(orm.NewCondition().And("task_id__in", id_list))
		if len(name_list) > 0 {
			cond = cond.OrCond(orm.NewCondition().And("name__in", name_list))
		}
	} else if len(name_list) > 0 {
		cond = cond.AndCond(orm.NewCondition().And("name__in", name_list))
	} else {
		return rs, nil
	}
	_, err := o.QueryTable(p).SetCond(cond).Limit(100).OrderBy("-id").All(&rs)
	if err != nil {
		return nil, err
	}
	return rs, nil

}

//这个更新可以避免获取最新的id
func (p *TaskLog) Update(o orm.Ormer, fields orm.Params) error {
	_, err := o.QueryTable(p).Filter("id", p.Id).Filter("task_id", p.TaskId).Update(fields)
	return err
}
