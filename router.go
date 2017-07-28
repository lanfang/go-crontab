package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	hd "github.com/lanfang/go-crontab/handlers"
	tp "github.com/lanfang/go-crontab/toplevel"
	"github.com/lanfang/go-lib/log"
	"time"
)

type CodoonApiResponse struct {
	status string      `json:"status"`
	data   interface{} `json:"data"`
	desc   string      `json:"description"`
}

func (this CodoonApiResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"status":      this.status,
		"data":        this.data,
		"description": this.desc,
	})
}

func startServer() {
	fmt.Printf("[%s]StartServer\n", time.Now().Format("2006-01-02 15:04:05"))

	router := gin.Default()

	router.Use(CodoonGetHeader)
	router.Use(HandleTimeLoger)

	order_router := router.Group("/crontab")
	{
		order_router.POST("/add_task", hd.HandleAddTask)                  //注册定时任务
		order_router.POST("/update_task", hd.HandleUpdateTask)            //更新定时任务的配置信息
		order_router.POST("/query_task_status", hd.HandleQueryTaskStatus) //查询定时任务结果
		order_router.POST("/run_again_task", hd.HandleRunAgain)           //重跑指定的任务

	}
	go hd.HandleServerRecover() //server 重启

	fmt.Println(router.Run(tp.G_Config.RPCListen))
}

func CodoonGetHeader(c *gin.Context) {
	r := c.Request
	log.Info("request url:%s; request header:%s", r.URL, r.Header)
}

func HandleTimeLoger(c *gin.Context) {
	start_time := time.Now()
	defer func() {
		log.Info("Handle[%s][cost:%dus]",
			c.Request.URL.Path, time.Now().Sub(start_time).Nanoseconds()/1000)
	}()

	c.Next()
}
