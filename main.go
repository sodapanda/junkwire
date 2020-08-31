package main

import (
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
	go tun.Read()
	time.Sleep(2 * time.Second)
	tun.Interrupt()
}
