package application

import (
	"fmt"
	"net"

	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/misc"
)

//AppServer server
type AppServer struct {
	conn       *net.UDPConn
	serverConn *connection.ServerConn
}

//NewAppServer new server
func NewAppServer(dstIP string, dstPort string, serverConn *connection.ServerConn) *AppServer {
	as := new(AppServer)
	address, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", dstIP, dstPort))
	misc.CheckErr(err)
	conn, err := net.DialUDP("udp4", nil, address)
	misc.CheckErr(err)
	as.conn = conn
	as.serverConn = serverConn
	return as
}

//Start start
func (as *AppServer) Start() {
	go as.socketToDevice()
	as.serverConn.AddHandler(handler{ser: as})
}

func (as *AppServer) socketToDevice() {
	readBuf := make([]byte, 2000)

	for {
		length, err := as.conn.Read(readBuf)
		misc.CheckErr(err)
		data := readBuf[:length]
		as.serverConn.Write(data, false)
	}
}

type handler struct {
	ser *AppServer
}

func (h handler) OnData(data []byte, conn *connection.ServerConn) {
	h.ser.conn.Write(data)
}

func (h handler) OnDisconnect() {

}
