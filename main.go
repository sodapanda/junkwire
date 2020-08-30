package main

import (
	"fmt"

	"github.com/sodapanda/junkwire/application"
	"github.com/sodapanda/junkwire/connection"
	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/device"
)

func main() {
	fmt.Println("good")
	device.Doit()
	connection.Conn()
	connection.ClientConn()
	application.AppSer()
	application.ClientApp()
	ds.NewBlockingQueue(1)
}
