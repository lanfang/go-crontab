//time wheel
//github.com/cloudwu/skynet/blob/master/skynet-src/skynet_timer.c
package timewhell

import (
	"container/list"
	"fmt"
	//"sync"
	bt "github.com/lanfang/go-crontab/basetype"
	//. "codoon_mall/crontabserver/toplevel"
)

const (
	TIME_NEAR_SHIFT  = 8
	TIME_NEAR        = 256
	TIME_LEVEL_SHIFT = 6
	TIME_LEVEL       = 64
	TIME_NEAR_MASK   = 255
	TIME_LEVEL_MASK  = 63
)

//record normal task position,
//--- slot(6bit + 6bit + 6bit + 6bit + 8bit) + levle(5bit)
type pos map[int64]int

type TimeWheel struct {
	near [TIME_NEAR]*list.List
	t    [4][TIME_LEVEL]*list.List
	//sync.Mutex
	time uint32
	quit chan struct{}
}

func New() *TimeWheel {
	t := new(TimeWheel)
	t.time = 0 //4294960000
	t.quit = make(chan struct{})

	var i, j int
	for i = 0; i < TIME_NEAR; i++ {
		t.near[i] = list.New()
	}

	for i = 0; i < 4; i++ {
		for j = 0; j < TIME_LEVEL; j++ {
			t.t[i][j] = list.New()
		}
	}

	return t
}

func (t *TimeWheel) CurTick() uint32 {
	return t.time
}

//this is Temporary...
func (t *TimeWheel) deleteNode(id int64, name string) {
	var i, j int
	var node *bt.Node
	for i = 0; i < TIME_NEAR; i++ {
		for e := t.near[i].Front(); e != nil; e = e.Next() {
			node = e.Value.(*bt.Node)
			if (node.Id == id || node.Name == name) && node.RunningType == bt.Normal {
				t.near[i].Remove(e)
				return
			}
		}
	}

	for i = 0; i < 4; i++ {
		for j = 0; j < TIME_LEVEL; j++ {
			for e := t.t[i][j].Front(); e != nil; e = e.Next() {
				node = e.Value.(*bt.Node)
				if (node.Id == id || node.Name == name) && node.RunningType == bt.Normal {
					t.t[i][j].Remove(e)
					return
				}
			}
		}
	}
}

func (t *TimeWheel) UpdateNode(n *bt.Node) {
	t.deleteNode(n.Id, n.Name)
	t.AddNode(n)
}

func (t *TimeWheel) AddNode(n *bt.Node) {
	//t.Lock()
	t.innerAddNode(n) //该接口要独立出来, 否则moveList死锁
	//t.Unlock()
}
func (t *TimeWheel) innerAddNode(n *bt.Node) {
	expire := n.ExpireTick //Relative t.time
	current := t.time
	if (expire | TIME_NEAR_MASK) == (current | TIME_NEAR_MASK) {
		t.near[expire&TIME_NEAR_MASK].PushBack(n)
	} else {
		var i uint32
		var mask uint32 = TIME_NEAR << TIME_LEVEL_SHIFT
		for i = 0; i < 3; i++ {
			if (expire | (mask - 1)) == (current | (mask - 1)) {
				break
			}
			mask <<= TIME_LEVEL_SHIFT
		}
		t.t[i][(expire>>(TIME_NEAR_SHIFT+i*TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK].PushBack(n)
	}
}

func (t *TimeWheel) String() string {
	return fmt.Sprintf("TimeWheel:time:%d", t.time)
}

func (t *TimeWheel) dispatchList(front *list.Element) {
	for e := front; e != nil; e = e.Next() {
		node := e.Value.(*bt.Node)
		go node.Run()
	}
}

func (t *TimeWheel) moveList(level, idx int) {
	vec := t.t[level][idx]
	front := vec.Front()
	vec.Init()
	for e := front; e != nil; e = e.Next() {
		node := e.Value.(*bt.Node)
		t.innerAddNode(node)
	}
}

func (t *TimeWheel) shift() {
	//mask =256
	var mask uint32 = TIME_NEAR
	t.time++
	ct := t.time
	if ct == 0 {
		t.moveList(3, 0)
	} else {
		//G_Logger.Info("afaf", t.time, ct)

		//TIME_NEAR_SHIFT=8
		time := ct >> TIME_NEAR_SHIFT
		var i int = 0
		for (ct & (mask - 1)) == 0 {
			idx := int(time & TIME_LEVEL_MASK)
			if idx != 0 {
				t.moveList(i, idx)
				break
			}
			mask <<= TIME_LEVEL_SHIFT
			time >>= TIME_LEVEL_SHIFT
			i++
		}
	}
}

func (t *TimeWheel) execute() {
	idx := t.time & TIME_NEAR_MASK
	vec := t.near[idx]
	if vec.Len() > 0 {
		front := vec.Front()
		vec.Init()
		t.dispatchList(front)
		return
	}
}

func (t *TimeWheel) Tick() {
	// try to dispatch timeout 0 (rare condition)
	t.execute()
	// shift time first, and then dispatch timer message
	t.shift()
	t.execute()
}

func (t *TimeWheel) Stop() {
	close(t.quit)
}

func (t *TimeWheel) DumpNodes() []*bt.Node {
	nodes := []*bt.Node{}
	var i, j int
	for i = 0; i < TIME_NEAR; i++ {
		for e := t.near[i].Front(); e != nil; e = e.Next() {
			nodes = append(nodes, e.Value.(*bt.Node))
		}
	}

	for i = 0; i < 4; i++ {
		for j = 0; j < TIME_LEVEL; j++ {
			for e := t.t[i][j].Front(); e != nil; e = e.Next() {
				nodes = append(nodes, e.Value.(*bt.Node))
			}
		}
	}
	return nodes
}
