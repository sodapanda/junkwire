package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/sodapanda/junkwire/codec"
	"github.com/sodapanda/junkwire/datastructure"
)

func TestInterlace(t *testing.T) {
	il := codec.NewInterlace(5, 1*time.Millisecond, func(dbf *datastructure.DataBuffer) {
		fmt.Print(dbf.Tag + " ")
	})
	go il.PushDown()

	for i := 0; i < 20; i++ {
		dbfs := make([]*datastructure.DataBuffer, 8)
		for j := 0; j < 8; j++ {
			dbf := datastructure.DataBuffer{
				Tag: fmt.Sprintf("row:%d_col:%d", i, j),
			}
			dbfs[j] = &dbf
		}
		il.Put(dbfs)
	}

	time.Sleep(100 * time.Second)
}
