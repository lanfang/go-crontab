package main

import (
	"fmt"
	"github.com/lanfang/go-crontab/crontab"
	tp "github.com/lanfang/go-crontab/toplevel"
	"os"
	"runtime"
	"strconv"
	"time"
)

func writePid() error {
	pid_fp, err := os.OpenFile(fmt.Sprintf("./%s.pid", tp.SERVERNAME), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open pid file failed[%s]\n", err)
		return err
	}
	defer pid_fp.Close()

	pid := os.Getpid()

	pid_fp.WriteString(strconv.Itoa(pid))
	return nil
}

func main() {
	//set runtime variable
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	//Init config, logger, db, perf
	tp.InitGlobal()

	//start server
	writePid()
	cron.New(time.Millisecond * 10) //tick interval is 0.01s
	//for perf
	go tp.StartHandleSignal()

	cron.Crontab.Start()

	startServer()
}
