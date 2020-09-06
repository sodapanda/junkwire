package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sodapanda/junkwire/application"
	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/device"
)

func main() {
	fmt.Println("start")

	fIsServer := flag.Bool("s", false, "server")
	flag.Parse()
	isServer := *fIsServer

	if isServer {
		server()
	} else {
		client()
	}
}

func client() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	client := application.NewAppClient("21007")
	client.Start()
	for {
		client.SetClientConn(nil)
		cc := connection.NewClientConn(tun, "10.1.1.2", "58.32.3.36", 8888, 10356)
		client.SetClientConn(cc)
		cc.WaitStop()
		fmt.Println("client stop restart")
		time.Sleep(5 * time.Second)
	}
}

func server() {
	tun := device.NewTunInterface("faketcp", "10.1.1.1", 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	sc := connection.NewServerConn("10.1.1.2", 10356, tun)
	sv := application.NewAppServer("127.0.0.1", "21007", sc)
	sv.Start()
	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println(sc)
}
