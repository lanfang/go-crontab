

## 添加定时任务
url(POST): http://localhost:6557/crontab/add_task

```
任务类型定义：const (
	TypeSingle = iota + 1	//单次任务, 启动时间固定, 可以为时间戳或者配置crontab
	TypeInterval			//间隔任务, 每隔多长长时间执行
	TypeHour				//时任务,每小时执行
	TypeDaily				//日任务,每天的某个时间执行
	TypeMonth				//月任务,每月的某个时间执行
	TypeYear				//年任务,每年的某个时间执行
	TypeWeek				//周任务,每周的某个时间执行
)

req:
{
	task_list:[
		{
			name:"任务名称",		//【必填】任务名称,每次添加任务需要保证名字唯一，否则添加失败
			enable: 1, 			//【必填】是否开启任务, 1 开启,2 停用
			module: 2, 			//【必填】任务所属模块，crontab sever指定
			type:1,				//【必填】任务类型,具体值建类型定义
			start_time,			//【必填】任务开始时间,可以是绝对时间和通配时间,绝对时间格式: 时间戳; 通配时间格式:秒,分,时,日,月,星期, 参考golang cron模块
			url:"xx",			//【必填】回调的接口, method为post,get时: http://addr:xx/path/aa; method为rpc时：addr:port/handler.function
			method:"post",		//【必填】回调方法post,get, rpc
			resp_regexp:"*",	//【选填】如果填了，当向url发送请求时，会通过正则验证返回结果是否正确: Match(resp_regexp, resp)
			body:"xxx", 	//【选填】body字符串，透传
			body_type:1,	//【选填】Content-Type: 1 Json, 2 Form， 如果不填默认json
			redo_param:"xxx"	//【选填】回调失败重试间隔, 用空格分割间隔时间，单位秒 ex: 2 4 8, 分别间隔2秒，4秒, 8秒 
		}
	],
}

//只有state为0的情况下才判断data的每一项
resp:
{
	  state:1， //返回码
	  msg:"xx", //错误信息
     data:{
			task_result:[ 返回task_id
				 {
				 	task_id:"xxx", //task_id
				 	name:"任务名称",
				 	err: "aaaa", //如果错误的话，会填入错误信息
				 }，
			] 
     }
}

```


## 修改定时任务的配置
url(POST): http://localhost:6557/crontab/update_task

```
如果设置了对应的字段，则执行更新操作，
task_id有效，则执行update xxx where task_id = task_id
task_id无效：则执行update xxx where name = name
优先选择task_id 更新， 如果task_id和name都无效，则不执行操作

req:
{
	task_list:[
		{
			task_id:12345,			// task_id和name字段必须选择填一个
			name:“定时改价”,		// task_id和name字段必须选择填一个
			//以下为需要更新value字段
			enable: 1, 			//【选填】是否开启任务, 1 开启,2 停用
			module: 2, 			//【选填】任务所属模块，crontab sever指定
			type:1,				//【选填】任务类型,具体值建类型定义
			start_time,			//【选填】任务开始时间,可以是绝对时间和通配时间,绝对时间格式: 时间戳; 通配时间格式:秒,分,时,日,月,星期, 参考golang cron模块
			url:"xx",			//【选填】回调的接口
			method:"post",		//【必填】回调方法
			resp_regexp:"*",	//【选填】如果填了，当向url发送请求时，会通过正则验证返回结果是否正确: Match(resp_regexp, resp)
			body:"xxx", 	//【选填】body字符串，透传
			body_type:1,	//【选填】Content-Type: 1 Json, 2 Form， 如果不填默认json
			redo_param:"xxx"	//【选填】回调失败重试间隔, 用空格分割间隔时间，单位秒 ex: 2 4 8, 分别间隔2秒，4秒, 8秒 
		}
	],
}

resp:
{
	  state:1， //返回码
	  msg:"xx", //错误信息
     data:{
			task_result:[ 返回请求里每一个task的更新结果
				 {
				 	task_id:"xxx", //task_id
				 	name:"任务名称",
				 	err: "aaaa", //如果错误的话，会填入错误信息
				 }，
			] 
     }
}
```

## 查询定时任务执行结果
url(POST): http://localhost:6557/crontab/query_task_status

```
优先根据task_id查询， 否则选择name查询
req:
{
	task_list:[
		{
			task_id:12345,			// 任务ID 优先选择id查询
			name:“定时改价”,		//任务名称
		}
	]
}

resp:
{
	  state:1， //返回码
	  msg:"xx", //错误信息
     data:{
     		log_list:[
		     {
		     	task_id:"xxx", //task_id
		     	name:"任务名称",
		     	run_type:1, //任务执行方式 1 正常启动，2 重试
		     	status:1, ////执行状态, 1 开始执行，2 执行成成功，3 执行失败
		     	begin_time:"sxx", //开始执行任务时间
		     	end_time:"xx", //结束执行时间
		     	url:"xxx", //任务访问的url
		     	body:“body内容”
		     	resp:"回包内容"
		     }，
	     ]
     } 
}
```

## 重跑定时任务
url(POST): http://localhost:6557/crontab/run_again_task

```
优先根据task_id查询， 否则选择name查询
req:
{
	task_list:[
		{
			task_id:12345,			// 任务ID 优先选择id查询
			name:“定时改价”,		//任务名称
		}
	],
}

resp:
{
	  state:1， //返回码
	  msg:"xx", //错误信息
     data:{
			task_result:[ 返回请求里每一个重跑结果
				 {
				 	task_id:"xxx", //task_id
				 	name:"任务名称",
				 	err: "aaaa", //如果错误的话，会填入错误信息
				 }，
			] 
     }
}
```

