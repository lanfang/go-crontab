// This library implements a cron spec parser and runner.  See the README for
// more details.
package cron

import (
	bt "github.com/lanfang/go-crontab/basetype"
	md "github.com/lanfang/go-crontab/models"
	timewheel "github.com/lanfang/go-crontab/timewheel"
	"github.com/lanfang/go-lib/log"
	"strconv"
	"strings"
	"time"
)

var Crontab *Cron

// Cron keeps track of any number of entries, invoking the associated func as
// specified by the schedule. It may be started, stopped, and the entries may
// be inspected while running.
type nodes []*bt.Node

type Cron struct {
	th        *timewheel.TimeWheel
	stop      chan struct{}
	add       chan *bt.Node
	update    chan *bt.Node
	remove    chan string
	node_list chan nodes
	running   bool
	tick      time.Duration
}

// New returns a new Cron job runner.
func New(t time.Duration) {
	if Crontab == nil {
		Crontab = &Cron{
			th:        timewheel.New(),
			add:       make(chan *bt.Node),
			update:    make(chan *bt.Node),
			remove:    make(chan string),
			stop:      make(chan struct{}),
			node_list: make(chan nodes),
			running:   false,
			tick:      t,
		}
	}
}

func (c *Cron) innerAddNode(node *bt.Node) {
	if !c.running {
		c.th.AddNode(node)
	} else {
		c.add <- node
	}
}

func (c *Cron) taskToNode(t *md.Task, node *bt.Node) error {
	now := time.Now().Unix()
	arg := new(bt.CallBackArg)
	node.Id = t.Id
	//node.Name = t.Name 任务名称暂时没使用,取消, 节省内存
	node.RetryCnt = 0
	if len(t.RedoParam) > 0 {
		kv := strings.Fields(t.RedoParam)
		for _, v := range kv {
			delay_seconds, _ := strconv.ParseInt(v, 10, 64)
			node.RetryList = append(node.RetryList, uint32(delay_seconds))
		}
	}
	var run_time int64 = 0
	if t.Type == bt.TypeSingle {
		node.Type = t.Type
		run_time, _ = strconv.ParseInt(t.StartTime, 10, 64)
	} else if t.Type > bt.TypeSingle {
		node.Type = t.Type
		node.Schedule = Parse(t.StartTime)
		run_time = node.Schedule.Next(time.Now().Local()).Unix()
	} else {
		//error, do someting return
		log.Error("task type err ", t)
		return bt.ErrTaskTypeInvalid
	}
	arg.NodeInfo = node
	node.Arg = arg

	var remain_time int64 = 0
	if run_time >= now {
		remain_time = run_time - now
	}
	node.ExpireTick = uint32((time.Second * time.Duration(remain_time) / c.tick)) + c.th.CurTick()
	return nil
}

// Add New Task
func (c *Cron) AddTask(t md.Task, r bt.RunType, fn interface{}) error {
	node := new(bt.Node)
	err := c.taskToNode(&t, node)
	node.Func = fn
	node.RunningType = r
	c.innerAddNode(node)
	return err
}

//Re Add Timer Node
func (c *Cron) AddRepeatTask(node *bt.Node) {
	node.RetryCnt = 0
	node.RunningType = bt.Normal
	now := time.Now()
	cur_seconds := now.Unix()
	run_seconds := node.Schedule.Next(now).Unix()
	var remain_seconds int64 = 0
	if run_seconds >= cur_seconds {
		remain_seconds = run_seconds - cur_seconds
	}
	node.ExpireTick = uint32((time.Second * time.Duration(remain_seconds) / c.tick)) + c.th.CurTick()
	c.innerAddNode(node)
}

//Retry Task
func (c *Cron) AddRetryTask(node *bt.Node, delay uint32) {
	node.RetryCnt++
	node.ExpireTick = uint32((time.Second * time.Duration(delay) / c.tick)) + c.th.CurTick()
	c.innerAddNode(node)
}

//slowly
func (c *Cron) UpdateTask(t md.Task, fn interface{}) error {
	node := new(bt.Node)
	err := c.taskToNode(&t, node)
	node.Func = fn
	node.RunningType = bt.Normal
	if !c.running {
		c.th.UpdateNode(node)
	} else {
		c.update <- node
	}
	return err
}

//This is not work
func (c *Cron) RemoveJob(name string) {
	if !c.running {

	}

	c.remove <- name
}

// Entries returns a snapshot of the cron entries.
func (c *Cron) AllTask() []*bt.Node {
	if c.running {
		c.node_list <- nil
		x := <-c.node_list
		return x
	}
	return c.nodeSnapshot()
}

// Start the cron scheduler in its own go-routine.
func (c *Cron) Start() {
	c.running = true
	go c.run()
}

// Run the scheduler.. this is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run() {
	// Figure out the next activation times for each entry.
	tick := time.NewTicker(c.tick)
	defer tick.Stop()

	for {
		now := time.Now()
		select {
		case <-tick.C:
			c.th.Tick()
		case node := <-c.add:
			c.th.AddNode(node)
		case node := <-c.update:
			c.th.UpdateNode(node)
		case name := <-c.remove:
			//do nothing
			log.Info("timer remove:", name)
		case <-c.node_list:
			c.node_list <- c.nodeSnapshot()

		case <-c.stop:
			log.Info("timer stop now=", now)
			return
		}
	}
}

// Stop the cron scheduler.
func (c *Cron) Stop() {
	c.stop <- struct{}{}
	c.running = false
}

// entrySnapshot returns a copy of the current cron entry list.
func (c *Cron) nodeSnapshot() []*bt.Node {
	return c.th.DumpNodes()
}
