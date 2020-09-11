package application

import (
	"net"

	"github.com/sodapanda/junkwire/codec"
	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/misc"
)

//AppClientFec client with fec
type AppClientFec struct {
	conn         *net.UDPConn
	connAddr     *net.UDPAddr
	clientConn   *connection.ClientConn
	rcv          int
	seg          int
	parity       int
	codec        *codec.FecCodec
	encodePool   *datastructure.DataBufferPool
	decodeResult []*datastructure.DataBuffer
}

//NewAppClientFec new
func NewAppClientFec(listenPort string, seg int, parity int, codec *codec.FecCodec) *AppClientFec {
	ac := new(AppClientFec)
	addr, err := net.ResolveUDPAddr("udp4", ":"+listenPort)
	misc.CheckErr(err)
	conn, err := net.ListenUDP("udp4", addr)
	misc.CheckErr(err)
	ac.conn = conn
	ac.seg = seg
	ac.parity = parity
	ac.codec = codec
	ac.encodePool = datastructure.NewDataBufferPool()
	ac.decodeResult = make([]*datastructure.DataBuffer, seg)
	for i := range ac.decodeResult {
		ac.decodeResult[i] = new(datastructure.DataBuffer)
		ac.decodeResult[i].Data = make([]byte, 2000)
	}
	return ac
}

//Start start
func (ac *AppClientFec) Start() {
	go ac.socketToDevice()
}

func (ac *AppClientFec) socketToDevice() {
	buffer := make([]byte, 2000)
	sb := codec.NewStageBuffer(ac.seg)
	fullDataBuffer := make([]byte, 2000*ac.seg)
	for {
		length, addr, err := ac.conn.ReadFromUDP(buffer)
		misc.CheckErr(err)
		ac.connAddr = addr
		data := buffer[:length]
		encodeResult := make([]*datastructure.DataBuffer, ac.seg+ac.parity)

		sb.Append(data, uint16(length), fullDataBuffer, ac.codec, func(cSb *codec.StageBuffer, resultData []byte, realLength int) {
			for i := range encodeResult {
				encodeResult[i] = ac.encodePool.PoolGet()
			}

			ac.codec.Encode(resultData, realLength, encodeResult)

			for i := range encodeResult {
				item := encodeResult[i]
				if ac.clientConn != nil {
					ac.clientConn.Write(item.Data[:item.Length], false)
				}
				ac.encodePool.PoolPut(item)
			}
		})
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

	rcvPkt := new(codec.FtPacket)
	rcvPkt.Decode(data)

	done := ch.ac.codec.Decode(rcvPkt, ch.ac.decodeResult)
	if !done {
		return
	}

	for _, d := range ch.ac.decodeResult {
		if d.Length == 0 {
			continue
		}
		_, err := ch.ac.conn.WriteToUDP(d.Data[:d.Length], ch.ac.connAddr)
		d.Length = 0 //设置为0 表示没有内容
		misc.CheckErr(err)
	}
}
func (ch clientFecHandler) OnDisconnect(cc *connection.ClientConn) {}
func (ch clientFecHandler) OnConnect(cc *connection.ClientConn)    {}
