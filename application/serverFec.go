package application

import (
	"fmt"
	"net"

	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/misc"
)

//AppServerFec AppServerFec
type AppServerFec struct {
	conn       *net.UDPConn
	serverConn *connection.ServerConn
}

//NewAppServerFec NewAppServerFec
func NewAppServerFec(dstIP string, dstPort string, serverConn *connection.ServerConn) *AppServerFec {
	as := new(AppServerFec)
	address, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", dstIP, dstPort))
	misc.CheckErr(err)
	conn, err := net.DialUDP("udp4", nil, address)
	misc.CheckErr(err)
	as.conn = conn
	as.serverConn = serverConn
	return as
}

//Start start
func (as *AppServerFec) Start() {
	go as.socketToDevice()
	as.serverConn.AddHandler(handlerFec{ser: as})
}

func (as *AppServerFec) socketToDevice() {
	readBuf := make([]byte, 2000)

	for {
		length, err := as.conn.Read(readBuf)
		misc.CheckErr(err)
		data := readBuf[:length]
		as.serverConn.Write(data, false)
	}
}

type handlerFec struct {
	ser *AppServerFec
}

func (h handlerFec) OnData(data []byte, conn *connection.ServerConn) {
	h.ser.conn.Write(data)
}

func (h handlerFec) OnDisconnect() {

}
