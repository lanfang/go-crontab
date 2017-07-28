package handlers

import (
	md "github.com/lanfang/go-crontab/models"
)

type Result struct {
	Name   string `json:"name`
	TaskId int64  `json:"task_id`
	Err    error  `json:"err`
}

type TaskInfo struct {
	TaskId int64  `json:"task_id"`
	Name   string `json:"name"`
}

//增加定时任务
type AddTaskReq struct {
	TaskList []md.Task `json:"task_list"`
}

type AddTaskResp struct {
	ResultList []Result `json:"task_result"`
}

//更新定时任务
type UpdateTaskReq struct {
	TaskList []md.Task `json:"task_list"`
}

type UpdateTaskResp struct {
	ResultList []Result `json:"task_result"`
}

//查询执行状态
type QueryLogArg struct {
	TaskList []TaskInfo `json:"task_list"`
}

type QueryLogResp struct {
	TaskLog []md.TaskLog `json:"log_list"`
}

//重跑
type RunAgainReq struct {
	TaskList []TaskInfo `json:"task_list"`
}

type RunAgainResp struct {
	ResultList []Result `json:"task_result"`
}
