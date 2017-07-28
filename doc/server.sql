--创建定时服务DB
create database timer_db;

--任务配置表
CREATE TABLE timer_db.task_config_info (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  name varchar(64) NOT NULL,
  enable tinyint(4) NOT NULL DEFAULT '1' COMMENT '是否开启任务: 1 开启;2 不开启',
  module tinyint(4) NOT NULL DEFAULT '0' COMMENT '该任务属于哪个模块',
  type tinyint(4) NOT NULL DEFAULT '0' COMMENT '任务类型(单次任务，间隔任务，时任务，日任务，月任务等)',
  start_time varchar(32) NOT NULL DEFAULT '' COMMENT '任务执行时间，JSON格式：秒,分,时,日,月,星期',
  url varchar(512) NOT NULL DEFAULT '' COMMENT '请求的接口',
  method varchar(32) NOT NULL DEFAULT '' COMMENT '请求方法',
  resp_regexp varchar(32) NOT NULL DEFAULT '' COMMENT 'url回包正则，调用时用于判断是否为预期回包',
  body text COMMENT 'body参数,无脑透传',
  body_type tinyint(4) NOT NULL DEFAULT '0' COMMENT 'Content-Type: 1 Json, 2 Form',
  redo_param varchar(32) NOT NULL DEFAULT '' COMMENT '重试参数：JSON格式,重试次数，间隔等',
  status tinyint(4) NOT NULL DEFAULT '0' COMMENT '1 开始执行，2 执行成成功，3 执行失败',
  begin_time datetime DEFAULT '0000-00-00 00:00:00' COMMENT '开始执行时间',
  PRIMARY KEY (id),
  UNIQUE KEY I_name (name)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;

--任务日志表
CREATE TABLE timer_db.task_log (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  task_id bigint(20) NOT NULL DEFAULT '0' COMMENT '任务ID',
  name varchar(32) NOT NULL DEFAULT '' COMMENT '名称',
  module tinyint(4) NOT NULL DEFAULT '0' COMMENT '该任务属于哪个模块',
  running_type tinyint(4) NOT NULL DEFAULT '0' COMMENT '1 正常启动，2 重试',
  status tinyint(4) NOT NULL DEFAULT '0' COMMENT '1 开始执行，2 执行成成功，3 执行失败',
  begin_time datetime NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT '开始执行时间',
  end_time datetime DEFAULT '0000-00-00 00:00:00' COMMENT '结束执行时间',
  url varchar(512) NOT NULL DEFAULT '' COMMENT '请求的接口',
  method varchar(32) NOT NULL DEFAULT '' COMMENT '请求方法',
  body text COMMENT 'body参数,透传',
  resp text COMMENT 'url回包',
  PRIMARY KEY (id),
  KEY I_task_id (task_id),
  KEY I_name (name),
  KEY index_begin_time (begin_time)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;



