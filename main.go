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

	sc := connection.NewServerConn("10.1.1.2", 8888, tun, serverHandler{})

	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println(sc)
}

type serverHandler struct {
	name string
}

func (sh serverHandler) OnData([]byte) {
	fmt.Println("on data")
}

func (sh serverHandler) OnDisconnect() {
	fmt.Println("disconnect")
}
