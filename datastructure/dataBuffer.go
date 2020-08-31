package datastructure

import "sync"

//DataBuffer byte slice and length,从pool里取出来，然后装入不同长度的内容之后放入队列
type DataBuffer struct {
	Data   []byte
	Length int
}

type DataBufferPool struct {
	dataPool sync.Pool
}

func NewDataBufferPool() *DataBufferPool {
	pool := new(DataBufferPool)
	pool.dataPool = sync.Pool{
		New: func() interface{} {
			data := new(DataBuffer)
			data.Data = make([]byte, 2000)
			return data
		},
	}
	return pool
}

func (dp *DataBufferPool) PoolGet() *DataBuffer {
	item := dp.dataPool.Get()
	return item.(*DataBuffer)
}

func (dp *DataBufferPool) PoolPut(item *DataBuffer) {
	dp.dataPool.Put(item)
}
