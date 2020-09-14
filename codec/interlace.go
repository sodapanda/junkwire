package codec

import (
	"fmt"
	"sync"
	"time"

	"github.com/sodapanda/junkwire/datastructure"
)

type row struct {
	datas  []*datastructure.DataBuffer
	cursor int //下一个要发的index
	runOut bool
}

//Interlace 交织
type Interlace struct {
	table    []*row
	size     int //有数据的行数
	cap      int //最多几行
	callback func(dbf *datastructure.DataBuffer)
	lock     *sync.Mutex
	notFull  *sync.Cond
	notEmpty *sync.Cond
	interval time.Duration //多久发完一组包
	stopFlag bool
}

//NewInterlace new
func NewInterlace(cap int, interval time.Duration, callback func(dbf *datastructure.DataBuffer)) *Interlace {
	il := new(Interlace)
	il.table = make([]*row, cap)
	il.size = 0
	il.cap = cap
	il.callback = callback
	il.lock = new(sync.Mutex)
	il.notFull = sync.NewCond(il.lock)
	il.notEmpty = sync.NewCond(il.lock)
	il.interval = interval
	for i := range il.table {
		item := new(row)
		item.runOut = true
		il.table[i] = item
	}
	return il
}

//Put put
func (il *Interlace) Put(intpuData []*datastructure.DataBuffer) {
	il.lock.Lock()

	for il.size == il.cap {
		il.notFull.Wait()
	}

	for _, thisRow := range il.table {
		if thisRow.runOut {
			thisRow.datas = intpuData
			thisRow.cursor = 0
			thisRow.runOut = false
			il.size = il.size + 1
			break
		}
	}
	il.lock.Unlock()
	il.notEmpty.Signal()
}

//PushDown push down to device
func (il *Interlace) PushDown() {
	for !il.stopFlag {
		il.lock.Lock()
		for il.size == 0 {
			il.notEmpty.Wait()
		}

		//把每一行的cursor向前推进(调用回调)，到头了就标记到头了
		for _, row := range il.table {
			if row.runOut {
				continue
			}
			data := row.datas[row.cursor]
			il.callback(data)
			row.cursor = row.cursor + 1
			if row.cursor == len(row.datas) {
				row.runOut = true
				il.size--
				il.notFull.Signal()
			}
		}
		il.lock.Unlock()
		time.Sleep(il.interval)
	}
}

//Dump debug
func (il *Interlace) Dump() {
	for i := range il.table {
		row := il.table[i]
		for _, item := range row.datas {
			fmt.Print(item.Tag + " ")
		}
		fmt.Print("\n")
	}
}
