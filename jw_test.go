package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/sodapanda/junkwire/codec"
	"github.com/sodapanda/junkwire/datastructure"
)

func TestInterlace(t *testing.T) {
	//想1秒发完一组包 一组包有4个 1000ms/4 = 250ms 发一次
	il := codec.NewInterlace(10, 500*time.Millisecond, func(dbf *datastructure.DataBuffer) {
		fmt.Print(time.Now().UnixNano() / 1000000)
		fmt.Print(dbf.Tag)
	})

	go il.PushDown()

	for i := 0; i < 50; i++ {
		//有10行
		dbfs := make([]*datastructure.DataBuffer, 4)
		//每行4个
		for j := 0; j < 4; j++ {
			dbf := new(datastructure.DataBuffer)
			if i%9 == 0 {
				dbf.Tag = fmt.Sprintf("row:%d_col:%d\n", i, j)
			} else {
				dbf.Tag = fmt.Sprintf("row:%d_col:%d ", i, j)
			}
			dbfs[j] = dbf
		}
		il.Put(dbfs)
	}

	time.Sleep(100 * time.Second)
}
