package datastructure

import (
	"container/list"
	"time"

	"sync"

	"github.com/sodapanda/junkwire/misc"
)

//BlockingQueue 阻塞队列
type BlockingQueue struct {
	lock      *sync.Mutex
	capacity  int
	notEmpty  *sync.Cond
	notFull   *sync.Cond
	dataList  *list.List
	interrupt bool
	size      int
}

//NewBlockingQueue 创建队列
func NewBlockingQueue(capacity int) *BlockingQueue {
	q := new(BlockingQueue)
	q.lock = new(sync.Mutex)
	q.capacity = capacity
	q.notEmpty = sync.NewCond(q.lock)
	q.notFull = sync.NewCond(q.lock)
	q.dataList = list.New()
	q.interrupt = false
	q.size = 0
	return q
}

//Put put item,block if full
func (q *BlockingQueue) Put(data *DataBuffer) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for q.size == q.capacity && !q.interrupt {
		q.notFull.Wait()
	}
	if q.interrupt {
		q.interrupt = false
		misc.PLog("return from interrupted Put")
		return
	}
	q.dataList.PushBack(data)
	q.size++
	q.notEmpty.Signal()
}

//Get item block if empty,return nil if interrupted
func (q *BlockingQueue) Get() *DataBuffer {
	q.lock.Lock()
	defer q.lock.Unlock()

	for q.size == 0 && !q.interrupt {
		q.notEmpty.Wait()
	}
	if q.interrupt {
		q.interrupt = false
		return nil
	}
	element := q.dataList.Back()
	rst := element.Value.(*DataBuffer)
	q.size--
	q.dataList.Remove(element)
	q.notFull.Signal()
	return rst
}

//Interrupt stop
func (q *BlockingQueue) Interrupt() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.interrupt = true
	q.notEmpty.Signal()
	q.notFull.Signal()
	misc.PLog("interrupt called")
}

//GetWithTimeout Get item,block with given time
func (q *BlockingQueue) GetWithTimeout(timeout time.Duration) *DataBuffer {
	q.lock.Lock()
	defer q.lock.Unlock()
	tm := time.NewTimer(timeout)
	chann := make(chan int, 1)
	isTimeout := false

	go func() {
		for q.size == 0 && !isTimeout {
			q.notEmpty.Wait()
		}
		chann <- 1
	}()

	select {
	case <-chann:
		element := q.dataList.Back()
		rst := element.Value.(*DataBuffer)
		q.size--
		q.dataList.Remove(element)
		q.notFull.Signal()
		defer tm.Stop()
		return rst
	case <-tm.C:
		isTimeout = true
		q.notEmpty.Signal()
		<-chann
		return nil
	}
}

//GetSize get current size
func (q *BlockingQueue) GetSize() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.size
}
