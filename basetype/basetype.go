package basetype

import (
	"time"
	//"fmt"
	"errors"
	"reflect"
)

const (
	RetSysError int = iota - 1
	RetOk
	RetInvalidParam
	RetTypeInvalid
	RetStartTimeInvalid
	RetCallBackInvalid
	RetIdAndNameInvalid
)

var (
	ErrSysError             = errors.New("system error")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrTaskTypeInvalid      = errors.New("task type is invalid")
	ErrTaskStartTimeInvalid = errors.New("task starttime is invalid")
	ErrTaskCallBackInvalid  = errors.New("task url or methon is invalid")
	ErrIdAndNameInvalid     = errors.New("id and name is invalid")
)

var CodeToMsg map[int]error = map[int]error{
	RetInvalidParam:     ErrInvalidArgument,
	RetTypeInvalid:      ErrTaskTypeInvalid,
	RetStartTimeInvalid: ErrTaskStartTimeInvalid,
	RetCallBackInvalid:  ErrTaskCallBackInvalid,
	RetIdAndNameInvalid: ErrIdAndNameInvalid,
}

var NoBodyMethods map[string]bool = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"DELETE":  true,
	"OPTIONS": true,
}

type RunType int

const (
	Normal RunType = iota + 1
	Retry
	DoAgain
)

type EnableType int

const (
	Enable EnableType = iota + 1
	Disable
)

type StatusType int

const (
	Begin StatusType = iota + 1
	Success
	Fail
)

type NodeType int

const (
	TypeSingle   NodeType = iota + 1 //单次任务, 启动时间固定, 可以为时间戳或者配置crontab
	TypeInterval                     //间隔任务, 每隔多长长时间执行
	TypeHour                         //时任务,每小时执行
	TypeDaily                        //日任务,每天的某个时间执行
	TypeMonth                        //月任务,每月的某个时间执行
	TypeYear                         //年任务,每年的某个时间执行
	TypeWeek                         //周任务,每周的某个时间执行

	InnerTypeClearLog = 100 //定期清理日志
)

type ContentType int

const (
	ContentJson ContentType = iota + 1
	ContentForm
)

/*
// Job is an interface for submitted cron jobs.
type Job interface {
	Run()
}
*/
// The Schedule describes a job's duty cycle.
type Schedule interface {
	// Return the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

type Node struct {
	// Unique Id to identify, this is task id.
	Id          int64
	Name        string
	RunningType RunType //normal, retry, doagain

	//Timer callback
	Func interface{}

	Arg interface{}

	//Current retry count
	RetryCnt int

	//Retry list second
	RetryList []uint32

	Type NodeType

	// If type is Repeate, Schedule is valid, the schedule on which this job should be run.
	Schedule Schedule

	// If type is Single, Next is valid ,the next time the job will run.
	//Next time.Time

	//expire tick, real run after ExpireTick
	ExpireTick uint32
}

func (n *Node) Run() {
	//fmt.Println("begin run", n)
	params := make([]reflect.Value, 0, 1)
	params = append(params, reflect.ValueOf(n.Arg))
	f := reflect.ValueOf(n.Func)
	f.Call(params)
}

func (n *Node) IsNormalRun() bool {
	return n.RetryCnt == 0
}

func (n *Node) IsNeedRetry() bool {
	return n.RetryCnt < len(n.RetryList)
}

func (n *Node) Copy(s *Node) {
	n.Id = s.Id
	n.Name = s.Name
	n.RunningType = s.RunningType
	n.Func = s.Func
	n.RetryCnt = s.RetryCnt
	for _, v := range s.RetryList {
		n.RetryList = append(n.RetryList, v)
	}
	n.Type = s.Type
	n.Schedule = s.Schedule
	n.ExpireTick = s.ExpireTick

}

//Timer Func call back arg
type CallBackArg struct {
	NodeInfo *Node
}

const (
	MethodRpc = "RPC"
)
