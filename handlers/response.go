package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

type Response struct {
	State int         `json:"state"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

func ReadRequestJson(c *gin.Context, obj interface{}) error {
	var body []byte
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, obj); err != nil {
		return err
	}
	fmt.Println(string(body), obj)
	return nil
}

func WriteRespons(c *gin.Context, code int, msg string, data interface{}) {
	ret_data_buf, _ := json.Marshal(data)
	c.JSON(http.StatusOK, Response{
		State: code,
		Msg:   msg,
		Data:  string(ret_data_buf),
	})
}
