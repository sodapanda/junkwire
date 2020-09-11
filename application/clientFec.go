package application

import (
	"net"

	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/misc"
)

//AppClientFec client with fec
type AppClientFec struct {
	conn       *net.UDPConn
	connAddr   *net.UDPAddr
	clientConn *connection.ClientConn
	rcv        int
}

//NewAppClientFec new
func NewAppClientFec(listenPort string) *AppClientFec {
	ac := new(AppClientFec)
	addr, err := net.ResolveUDPAddr("udp4", ":"+listenPort)
	misc.CheckErr(err)
	conn, err := net.ListenUDP("udp4", addr)
	misc.CheckErr(err)
	ac.conn = conn
	return ac
}

//Start start
func (ac *AppClientFec) Start() {
	go ac.socketToDevice()
}

func (ac *AppClientFec) socketToDevice() {
	buffer := make([]byte, 2000)
	for {
		length, addr, err := ac.conn.ReadFromUDP(buffer)
		misc.CheckErr(err)
		ac.connAddr = addr
		data := buffer[:length]
		if ac.clientConn != nil {
			ac.clientConn.Write(data, false)
		}
	}
}

//SetClientConn set client connection
func (ac *AppClientFec) SetClientConn(clientConn *connection.ClientConn) {
	ac.clientConn = clientConn
	if clientConn != nil {
		ac.clientConn.AddHandler(clientFecHandler{ac: ac})
	}
}

type clientFecHandler struct {
	ac *AppClientFec
}

func (ch clientFecHandler) OnData(data []byte) {
	ch.ac.rcv++
	_, err := ch.ac.conn.WriteToUDP(data, ch.ac.connAddr)
	misc.CheckErr(err)
}
func (ch clientFecHandler) OnDisconnect(cc *connection.ClientConn) {}
func (ch clientFecHandler) OnConnect(cc *connection.ClientConn)    {}
