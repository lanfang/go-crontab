 a timer server, written with golang, similar to linux crontab

# go-crontab 介绍

1、实现类似crontab任务配置功能(感谢[@robfig](https://github.com/robfig/cron)),基于时间轮存储任务,同时任务落地到mysql
2、任务:一个http回调请求(注册任务时带上url以及参数)
3、支持批量注册、更新、重跑任务,查看任务执行状态,退步重试,结果校验等功能
# 使用

- 启动定时任务
定时服务依赖mysql做任务配置持久化, 先进行库表创建，[库表操作](https://github.com/lanfang/go-crontab/blob/development/doc/server.sql)

- 注册定时任务
  客户端向定时服务发起任务注册，[协议文档](https://github.com/lanfang/go-crontab/blob/development/doc/api.md)


# TODO
1、支持RPC任务
2、解决定时服务的单点,数据中心问题 [HA方案](https://github.com/lanfang/go-crontab/blob/development/doc/ha.md)


