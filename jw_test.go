package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/sodapanda/junkwire/codec"
)

func TestInterlace(t *testing.T) {
	seg := 5
	icodec := codec.NewFecCodec(seg, 10, 100)
	rstBf := make([]byte, 2000*seg)
	sb := codec.NewStageBuffer(icodec, seg, rstBf, 1*time.Second, func(sb *codec.StageBuffer, rst []byte, realLen int) {
		fmt.Println("callback ", realLen)
	})

	//1
	content := make([]byte, 10)
	sb.Append(content, uint16(10))

	//2
	content = make([]byte, 10)
	sb.Append(content, uint16(10))

	//3
	content = make([]byte, 10)
	sb.Append(content, uint16(10))

	//4
	content = make([]byte, 10)
	sb.Append(content, uint16(10))

	//5
	content = make([]byte, 10)
	sb.Append(content, uint16(10))

	//6
	content = make([]byte, 11)
	sb.Append(content, uint16(11))

	time.Sleep(100 * time.Second)
}
