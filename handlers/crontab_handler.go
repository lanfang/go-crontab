package handlers

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/orm"
	"github.com/gin-gonic/gin"
	bt "github.com/lanfang/go-crontab/basetype"
	cron "github.com/lanfang/go-crontab/crontab"
	md "github.com/lanfang/go-crontab/models"
	"github.com/lanfang/go-lib/http_client_cluster"
	"github.com/lanfang/go-lib/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//重启时,判断任务是否需要重跑,目前只判断一次性任务,避免运行多次
func IsTaskNeedRun(task md.Task) bool {
	is_run := true
	switch task.Type {
	case bt.TypeSingle:
		start_time, _ := strconv.ParseInt(task.StartTime, 10, 64) //配置的时间
		beg := task.BeginTime.Unix()
		if beg >= start_time {
			is_run = false
		}
	default:
		is_run = true
	}
	return is_run
}

func HandleServerRecover() {

	InitInnerCron()
	run_type := bt.Normal
	task := md.Task{}
	o := orm.NewOrm()

	task_list, err := task.FindEnableTask(o)
	if (err != nil) || (len(task_list) == 0) {
		log.Info("Invalid task list task_list:%+v, err:%+v", task_list, err)
		return
	}

	for _, task := range task_list {
		//重启后, 重新添加定时任务;
		//如果是单次任务,并且当日已执行, 则不添加
		if IsTaskNeedRun(task) {
			cron.Crontab.AddTask(task, run_type, TimerHandler)
		}
	}
}

//rpc调用逻辑没有暂时没抽出来
func dealRpcTask(task *md.Task) (string, error) {
	/*
		log.Info("dealRpcTask Begin id:%s, url:%s, body:%s", task.Id, task.Url, task.Body)
		//url:	addr:port/handler.function
		resp := make(map[string]interface{})
		var err error = nil
		for loop := true; loop; loop = false {
			rpc_info := strings.Split(task.Url, "/")
			if len(rpc_info) != 2 {
				err = fmt.Errorf("rpc url格式错误 %s", task.Url)
				log.Info("dealRpcTask ", err.Error())
				break
			}
			client := DefaultRpc.Get(rpc_info[0])
			if client == nil {
				err = fmt.Errorf(" rpc cleint is nil", task.Url)
				log.Info("dealRpcTask ", err.Error())
				break
			}
			req := json.RawMessage(task.Body)
			err = client.CallTimeout(rpc_info[1], &req, &resp)
		}
		real_resp, err2 := json.Marshal(resp)
		log.Info("dealRpcTask End  id:%s, url:%s, body:%s, err:%+v, marshal_err:%+v", task.Id, task.Url, task.Body, err, err2)
		return string(real_resp), err
	*/
	return "", nil
}

func dealTask(task *md.Task) (string, error) {
	var err error = nil
	var response *http.Response
	var req *http.Request
	var body_data io.Reader
	var rsp []byte

	headers := make(map[string]string, 0)
	start_time := time.Now()

	request_url, err := url.Parse(task.Url)

	if err != nil {
		log.Error("url parse failed![url:%s][err:%s]",
			task.Url, err)
		goto Notice
	}

	if !bt.NoBodyMethods[task.Method] && len(task.Body) > 0 {
		if task.BodyType == bt.ContentJson {
			body_data = bytes.NewBuffer([]byte(task.Body))
			headers["Content-Type"] = "application/json"
		} else if task.BodyType == bt.ContentForm {
			m := make(map[string]interface{})
			decoder := json.NewDecoder(strings.NewReader(task.Body))
			decoder.UseNumber()
			err = decoder.Decode(&m)

			_data := url.Values{}
			for data_key, data_value := range m {
				_data.Set(data_key, fmt.Sprintf("%v", data_value))
			}
			body_data = strings.NewReader(_data.Encode())
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}
	req, err = http.NewRequest(strings.ToUpper(task.Method), request_url.String(), body_data)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if err != nil {
		log.Error("request url[%s] failed![err:%s]",
			request_url.String(), err)
		goto Notice
	}
	//response, err = http.DefaultClient.Do(req)
	response, err = http_client_cluster.HttpClientClusterDo(req)
	if err != nil {
		log.Error("request url[%s] failed![err:%s]",
			request_url.String(), err)
		goto Notice
	}

	defer response.Body.Close()

	rsp, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("read body failed![url:%s][err:%s]",
			request_url.String(), err)
		goto Notice
	}

	log.Info("%s %s %v, response: %+v", task.Method, request_url.String(), response.StatusCode, string(rsp))

	if response.StatusCode != 200 {
		log.Error("request url[%s] failed![code:%d]", request_url.String(),
			response.StatusCode)
		err = fmt.Errorf("request url[%s] failed![code:%d]", request_url.String(),
			response.StatusCode)
		goto Notice
	} else if task.RespRegexp != "" {
		var reg *regexp.Regexp
		var err2 error
		if reg, err2 = regexp.Compile(task.RespRegexp); err2 == nil {
			if reg.Match(rsp) {
				err = nil
			} else {
				err = fmt.Errorf("response format not match regexp", request_url.String(), task.RespRegexp, rsp)
			}
		} else {
			err = err2
		}
	}

Notice:
	if err == nil {
		log.Info("[status:success][url:%s][cost:%dus]", request_url.String(),
			time.Now().Sub(start_time).Nanoseconds()/1000)
	} else {
		log.Info("[status:error][url:%s][cost:%dus][err:%s]", request_url.String(),
			time.Now().Sub(start_time).Nanoseconds()/1000, err)
	}

	return string(rsp), err
}

func GetTaskInfo(o orm.Ormer, task_id int64) (md.Task, error) {
	task := md.Task{}
	if err := task.FindTaskByTaskId(o, task_id); err != nil {
		log.Error("Task Config FindTaskByTaskId ", err)
		return task, err
	}
	return task, nil
}

func RunTask(o orm.Ormer, r bt.RunType, task md.Task) error {
	//强制重跑任务不做enable检查
	if r != bt.DoAgain && task.Enable != bt.Enable {
		log.Error("Task Config task is not enable task:%+v ", task)
		return nil
	}

	begin := time.Now()

	task_log := md.TaskLog{
		TaskId: task.Id, Name: task.Name, Module: task.Module,
		Status: bt.Begin, BeginTime: begin, RunningType: r,
		Url: task.Url, Method: task.Method, Body: task.Body,
	}
	//Insert init log
	if err := task_log.Insert(o); err != nil {
		log.Error("Inert log task_log:%+v, err:%+v ", task_log, err)
	}

	//run task
	update_task, update_log := make(orm.Params, 0), make(orm.Params, 0)
	var resp string
	var result error
	if strings.ToUpper(task.Method) == bt.MethodRpc {
		resp, result = dealRpcTask(&task)
	} else {
		resp, result = dealTask(&task)
	}

	if result != nil {
		update_task["status"], update_log["status"] = bt.Fail, bt.Fail
		update_log["resp"] = result.Error()
	} else {
		update_task["status"], update_log["status"] = bt.Success, bt.Success
		update_log["resp"] = resp
	}
	update_task["begin_time"] = begin
	update_log["end_time"] = time.Now()

	//update log
	if result == nil && task.Type == bt.TypeSingle {
		//对于单次任务如果执行成功则直接删除配置
		task.DeleteByTaskId(o, task.Id)
	} else if err := task.UpdateByTaskId(o, task.Id, update_task); err != nil {
		log.Error("Update task task.Id:%+v, update_task:%+v, eer:%+v", task.Id, update_task, err)
	}

	if err := task_log.Update(o, update_log); err != nil {
		log.Error("Update task_log task.Id:%+v, update_log:%+v, err:%+v", task.Id, update_log, err)
	}
	return result
}

//重试
func RetryTask(n *bt.Node) {
	if n.IsNeedRetry() {
		n.RunningType = bt.Retry
		cron.Crontab.AddRetryTask(n, n.RetryList[n.RetryCnt])
	}
}

func deepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func TimerHandler(tr interface{}) error {
	arg, ok := tr.(*bt.CallBackArg)
	log.Info("begin run task node:%+v ", arg.NodeInfo)
	if !ok {
		return bt.ErrInvalidArgument
	}
	o := orm.NewOrm()

	//获取任务配置
	task, err := GetTaskInfo(o, arg.NodeInfo.Id)
	if err != nil {
		log.Info("get task arg.NodeInfo:%+v, err:%+v", arg.NodeInfo, err)
		return err
	}

	//先产生下一个node
	if task.Enable == bt.Enable && arg.NodeInfo.Type != bt.TypeSingle && arg.NodeInfo.RunningType == bt.Normal {
		//添加节点, 下次到期执行
		cron.Crontab.AddRepeatTask(arg.NodeInfo)
	}

	//执行任务
	result := RunTask(o, arg.NodeInfo.RunningType, task)

	if result != nil {
		//失败重试
		if arg.NodeInfo.RunningType == bt.Normal {
			//因为是存的指针,所以需要给重试单独创建节点
			retry_node := new(bt.Node)
			retry_node.Copy(arg.NodeInfo)
			arg := new(bt.CallBackArg)
			arg.NodeInfo = retry_node
			retry_node.Arg = arg

			RetryTask(retry_node)
		} else {
			RetryTask(arg.NodeInfo)
		}
	}

	log.Info("end run ask_id=%s", arg)
	return nil
}

func IsValidTask(task *md.Task) (bool, error) {
	var is_valid = true
	var err error = nil
	for loop := true; loop; loop = false {
		if len(task.Url) == 0 || len(task.Method) == 0 {
			is_valid, err = false, bt.ErrTaskCallBackInvalid
			break
		}
		if strings.ToUpper(task.Method) == bt.MethodRpc {
			rpc_info := strings.Split(task.Url, "/")
			if len(rpc_info) != 2 {
				is_valid, err = false, fmt.Errorf("rpc url格式错误 ex:[addr:port/handler.func], [%s]", task.Url)
				break
			}
		}
		if !(task.Type >= bt.TypeSingle && task.Type <= bt.TypeWeek) {
			is_valid, err = false, bt.ErrTaskTypeInvalid
			break
		}

		if len(task.StartTime) == 0 {
			is_valid, err = false, bt.ErrTaskStartTimeInvalid
			break
		}
	}
	return is_valid, err
}

func innerRespons(c *gin.Context, codePtr *int, errPtr *error, replyPtr *interface{}) {
	code, err, reply := *codePtr, *errPtr, *replyPtr
	if err == nil {
		WriteRespons(c, code, "", reply)
	} else {
		WriteRespons(c, code, err.Error(), reply)
	}

}

//http handler,都是正常添加
func HandleAddTask(c *gin.Context) {
	log.Info("[+] HandleAddTask begin")
	run_type := bt.Normal
	var err error = nil
	code := bt.RetOk //default
	var reply interface{}
	defer innerRespons(c, &code, &err, &reply)

	req := AddTaskReq{}
	if err = ReadRequestJson(c, &req); err != nil {
		code = bt.RetInvalidParam
		log.Error("[-] err,", err)
		return
	}
	log.Info("[+] HandleAddTask req:%+v", req)

	if len(req.TaskList) == 0 {
		return
	}
	o := orm.NewOrm()

	batch_insert := func(req_list []md.Task) []Result {
		task_name_list, result_list := make([]string, 0), make([]Result, 0)
		name_to_task := make(map[string]md.Task)
		valid_list := make([]md.Task, 0)
		for _, task := range req_list {
			is_valid_task, err := IsValidTask(&task)
			if is_valid_task && err == nil {
				task_name_list = append(task_name_list, task.Name)
				name_to_task[task.Name] = task
				valid_list = append(valid_list, task)
			} else {
				log.Error("[-] HandleAddTasktask config task:%+v, err:%+v", task, err)
			}
		}
		if len(valid_list) == 0 {
			return result_list
		}
		_, err := o.InsertMulti(len(valid_list), &valid_list)
		if err == nil {
			name_id_map := make(orm.Params)
			sql := "select name, id from task_config_info "
			sql += fmt.Sprintf("WHERE name in (%s) ", strings.TrimSuffix(strings.Repeat("?,", len(task_name_list)), ","))
			if _, err := o.Raw(sql).SetArgs(task_name_list).RowsToMap(&name_id_map, "name", "id"); err != nil {
				log.Error("[-] HandleAddTask get task task_name_list:%+v, err:%+v", task_name_list, err)
			}
			for name, id := range name_id_map {
				tmp_id, _ := strconv.ParseInt(id.(string), 10, 64)
				tmp := Result{Name: name, TaskId: tmp_id}
				success_task := name_to_task[name]
				success_task.Id = tmp_id
				tmp.Err = cron.Crontab.AddTask(success_task, run_type, TimerHandler)
				result_list = append(result_list, tmp)
			}
		} else {
			log.Info("[-] HandleAddTask req_list:%+v, err:%+v", req_list, err)
		}
		return result_list
	}
	real_reply := AddTaskResp{}
	step_size := 300
	task_size := len(req.TaskList)
	for begin, end := 0, 0; begin < task_size; {
		end = begin + step_size
		if end > task_size {
			end = task_size
		}
		tmp_result := batch_insert(req.TaskList[begin:end])
		real_reply.ResultList = append(real_reply.ResultList, tmp_result...)
		begin = end
	}
	reply = real_reply
	log.Info("[-] HandleAddTask end reply:%+v", reply)
}

func GetUpdateField(task *md.Task) (fields orm.Params, is_update_th bool, err error) {
	is_update_th = false
	fields = make(orm.Params, 0)
	if task.Id == 0 && len(task.Name) == 0 {
		err = bt.ErrIdAndNameInvalid
		return
	}

	if task.Enable > 0 {
		fields["enable"] = task.Enable
		if task.Enable == bt.Enable {
			is_update_th = true
		}
	}
	if len(task.StartTime) > 0 {
		fields["start_time"] = task.StartTime
		is_update_th = true
	}

	if len(task.Url) > 0 {
		fields["url"] = task.Url

	}

	if len(task.Method) > 0 {
		fields["method"] = task.Method

	}

	if len(task.RespRegexp) > 0 {
		fields["resp_regexp"] = task.RespRegexp

	}

	if len(task.Body) > 0 {
		fields["body"] = task.Body

	}

	if task.BodyType > 0 {
		fields["body_type"] = task.BodyType

	}

	if len(task.RedoParam) > 0 {
		fields["redo_param"] = task.RedoParam
	}
	err = nil
	return
}

//目的废弃旧的taskid,再新增一个
func DisableAndUpdateTask(o orm.Ormer, old_task *md.Task, update_filed *orm.Params) (int64, error) {
	var err error = nil
	var new_task_id int64 = 0
	if err := o.Begin(); err != nil {
		log.Error("DB begin transaction failed: %s", err.Error())
		return new_task_id, err
	}

	for loop := true; loop; loop = false {
		if err = old_task.DeleteByTaskId(o, old_task.Id); err != nil {
			break
		}
		old_task.Id = 0 //db generate new task_id
		if new_task_id, err = old_task.Insert(o); err != nil {
			break
		}
		if err = old_task.UpdateByTaskId(o, new_task_id, *update_filed); err != nil {
			break
		}
	}

	if err == nil {
		if err := o.Commit(); err != nil {
			o.Rollback()
			log.Error("DB commit transaction failed: %s", err.Error())
			return new_task_id, err
		} else {

		}
	} else {
		o.Rollback()
	}
	return new_task_id, err
}

func HandleUpdateTask(c *gin.Context) {
	log.Info("[+] begin")
	var err error = fmt.Errorf("")
	code := bt.RetOk //default
	var reply interface{}
	defer innerRespons(c, &code, &err, &reply)

	req := UpdateTaskReq{}
	if err = ReadRequestJson(c, &req); err != nil {
		code = bt.RetInvalidParam
		log.Error("[-] err,", err)
		return
	}
	log.Info("req:%+v", req)
	real_reply := UpdateTaskResp{}
	o := orm.NewOrm()
	var update_field orm.Params
	var is_update_th bool = false
	for _, task := range req.TaskList {
		if update_field, is_update_th, err = GetUpdateField(&task); err == nil && len(update_field) > 0 {
			if is_update_th {
				//如果修改启动时间则需要修改时间轮
				old_task := md.Task{}
				if task.Id > 0 {
					err = old_task.FindTaskByTaskId(o, task.Id)
				} else if len(task.Name) > 0 {
					err = old_task.FindTaskByName(o, task.Name)
				} else {
					err = bt.ErrIdAndNameInvalid
					log.Error("[+] update task:", err, task)
				}
				if err == nil {
					//将旧的任务废弃,生成一个新任务
					var new_id int64
					if new_id, err = DisableAndUpdateTask(o, &old_task, &update_field); err == nil {
						new_task := md.Task{}
						if err = new_task.FindTaskByTaskId(o, new_id); err == nil {
							err = cron.Crontab.AddTask(new_task, bt.Normal, TimerHandler)
							task.Id = new_id
						}
					}
				}
			} else {
				//不需要修改时间轮,直接操作DB
				if task.Id > 0 {
					err = task.UpdateByTaskId(o, task.Id, update_field)
				} else if len(task.Name) > 0 {
					err = task.UpdateByName(o, task.Name, update_field)
				} else {
					err = bt.ErrIdAndNameInvalid
					log.Error("[+] update task:", err, task)
				}
			}
			/*
				if err == nil && is_update_th {
					if task.Id > 0 {
						err = task.FindTaskByTaskId(o, task.Id)
					} else if len(task.Name) > 0 {
						err = task.FindTaskByName(o, task.Name)
					}
					if err == nil {
						cron.Crontab.UpdateTask(task, TimerHandler)
					}
				}
			*/
		}
		real_reply.ResultList = append(real_reply.ResultList, Result{
			TaskId: task.Id,
			Name:   task.Name,
			Err:    err,
		})

	}
	reply = real_reply
	log.Info("[-] end reply:%+v", reply)

}

func HandleQueryTaskStatus(c *gin.Context) {
	log.Info("[+] begin")
	var err error = fmt.Errorf("")
	code := bt.RetOk //default
	var reply interface{}

	defer innerRespons(c, &code, &err, &reply)

	//args := make([]bt.QueryLogArg, 0)
	req := QueryLogArg{}
	if err = ReadRequestJson(c, &req); err != nil {
		//err = bt.ErrInvalidArgument
		code = bt.RetInvalidParam
		log.Error("[-] err,", err)
		return
	}
	id_list, name_list := make([]int64, 0), make([]string, 0)
	for _, p := range req.TaskList {
		if p.TaskId > 0 {
			id_list = append(id_list, p.TaskId)
		} else if len(p.Name) > 0 {
			name_list = append(name_list, p.Name)
		} else {
			log.Info("[] err param", p)
		}
	}

	o := orm.NewOrm()
	task_log := md.TaskLog{}
	real_resp := QueryLogResp{}
	if resp, err := task_log.FindByTaskIdOrName(o, id_list, name_list); err == nil {
		real_resp.TaskLog = resp
	}
	reply = real_resp
	log.Info("[-] end ", id_list, name_list)

}

func HandleRunAgain(c *gin.Context) {
	log.Info("[+] begin")
	run_type := bt.DoAgain
	var err error = fmt.Errorf("")
	code := bt.RetOk //default
	var reply interface{}

	defer innerRespons(c, &code, &err, &reply)

	req := RunAgainReq{}
	if err = ReadRequestJson(c, &req); err != nil {
		code = bt.RetInvalidParam
		log.Error("[-] err,", err)
		return
	}
	log.Error("[-] runagain req:,", req)

	o := orm.NewOrm()
	task := md.Task{}
	real_reply := RunAgainResp{}

	for _, p := range req.TaskList {
		if p.TaskId > 0 {
			err = task.FindTaskByTaskId(o, p.TaskId)
		} else if len(p.Name) > 0 {
			err = task.FindTaskByName(o, p.Name)
		}

		if err == nil {
			//强制重跑当做一个一次性任务,立马执行
			task.Type = bt.TypeSingle
			task.StartTime = "0"
			cron.Crontab.AddTask(task, run_type, TimerHandler)
		}

		real_reply.ResultList = append(real_reply.ResultList, Result{
			TaskId: task.Id,
			Name:   task.Name,
			Err:    err,
		})
	}

	reply = real_reply
	log.Info("[-] end ", reply)

}
