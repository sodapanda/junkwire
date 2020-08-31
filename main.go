package main

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sodapanda/junkwire/device"
)

func main() {
	fmt.Println("start")
	test()
}

func test() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)
	dbf := tun.ReadTimeout(1 * time.Second)
	if dbf == nil {
		fmt.Println("time out")
		return
	}
	fmt.Println(hex.Dump(dbf.Data[:dbf.Length]))
}
