package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/device"
)

func main() {
	fmt.Println("start")

	fIsServer := flag.Bool("s", false, "server")
	flag.Parse()
	isServer := *fIsServer

	if isServer {
		testServer()
	} else {
		testClient()
	}
}

func testClient() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	for {
		cc := connection.NewClientConn(tun, "10.1.1.2", "58.32.3.36", 8888, 10356, clientHandler{})
		cc.WaitStop()
		fmt.Println("client stop restart")
		time.Sleep(5 * time.Second)
	}
}

func testServer() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	sc := connection.NewServerConn("10.1.1.2", 10356, tun, serverHandler{})

	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println(sc)
}

///
type clientHandler struct {
	name string
}

func (ch clientHandler) OnData(data []byte) {
	fmt.Println("on data ", string(data))
}

func (ch clientHandler) OnDisconnect(cc *connection.ClientConn) {
	fmt.Println("disconnect")
}

func (ch clientHandler) OnConnect(cc *connection.ClientConn) {
	fmt.Println("connect")
	cc.Write([]byte("hell0!"), false)
}

///
type serverHandler struct {
	name string
}

func (ch serverHandler) OnData(data []byte, sc *connection.ServerConn) {
	fmt.Println("on data ", string(data))
	sc.Write(data, false)
}

func (ch serverHandler) OnDisconnect() {
	fmt.Println("disconnect")

	testClient()
}
