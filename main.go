package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/device"
)

func main() {
	fmt.Println("start")
	test()
}

func test() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	cc := connection.NewClientConn(tun, "10.1.1.2", "192.168.8.39", 8888, 9900, clientHandler{})

	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println(cc)
}

type clientHandler struct {
	name string
}

func (ch clientHandler) OnData([]byte) {
	fmt.Println("on data")
}

func (ch clientHandler) OnDisconnect() {
	fmt.Println("disconnect")
}
