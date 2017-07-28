### 横向扩展
```
任务落地到mysql(同一张表，数据量实在太大，再考虑分表)， 每一个timer_woker既是worker，也是router
路由策略：fnvhash(task_id)%woker_num, 再pub消息到etcd
```
### 定时任务注册：
```
client(step_1) -> 任一time_woker(step_2) ->real_worker(时间轮)
step_1:客户端发起请求
step_2:数据落地，得到task_id; pub task_id到etcd key_1
step_3: timer_worker subscribe key_1， 获取到task_id, 注册到时间轮
```
### 高可用
```
将woker分组， 每一个组里面有一个master工作，多个slave;master挂掉以后，由其中一个slave接管任务。选主方案: 基于etcd实现选主机制
```