package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/sodapanda/junkwire/device"
)

// func TestQueue(t *testing.T) {
// 	q := ds.NewBlockingQueue(2)
// 	dataBuffer := new(ds.DataBuffer)
// 	q.Put(dataBuffer)
// 	fmt.Println("input 1 ,size ", q.GetSize())
// 	q.Put(dataBuffer)
// 	fmt.Println("input 1 ,size ", q.GetSize())
// 	go func() {
// 		fmt.Println("   try to input when full")
// 		q.Put(dataBuffer)
// 		fmt.Println("   input success", q.GetSize())
// 	}()

// 	time.Sleep(3 * time.Second)
// 	q.Get()
// 	fmt.Println("    output 1, size ", q.GetSize())

// 	q.Get()
// 	q.Get()
// 	fmt.Println("now empty try to get with timeout")
// 	q.GetWithTimeout(2 * time.Second)
// 	fmt.Println("time out !")
// 	time.Sleep(10 * time.Second)
// }

func TestTun(t *testing.T) {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)
	dbf := tun.Read()
	fmt.Println(hex.Dump(dbf.Data[:dbf.Length]))
}
