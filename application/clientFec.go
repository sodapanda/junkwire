package application

import (
	"fmt"
	"net"
	"time"

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
	seg          int //数据组个数
	parity       int //纠错组个数
	duration     int //交织时间段 毫秒
	codec        *codec.FecCodec
	encodePool   *datastructure.DataBufferPool
	decodeResult []*datastructure.DataBuffer
	il           *codec.Interlace //交织
}

//NewAppClientFec new
func NewAppClientFec(listenPort string, seg int, parity int, icodec *codec.FecCodec, duration int, rowCount int) *AppClientFec {
	ac := new(AppClientFec)
	addr, err := net.ResolveUDPAddr("udp4", ":"+listenPort)
	misc.CheckErr(err)
	conn, err := net.ListenUDP("udp4", addr)
	misc.CheckErr(err)
	ac.conn = conn
	ac.seg = seg
	ac.parity = parity
	ac.duration = duration
	ac.codec = icodec
	ac.encodePool = datastructure.NewDataBufferPool()
	ac.decodeResult = make([]*datastructure.DataBuffer, seg)
	for i := range ac.decodeResult {
		ac.decodeResult[i] = new(datastructure.DataBuffer)
		ac.decodeResult[i].Data = make([]byte, 2000)
	}

	inv := time.Duration((float32(duration) / float32(seg+parity)) * 1000)
	misc.PLog(fmt.Sprintf("interval %d", inv))

	ac.il = codec.NewInterlace(rowCount, inv*time.Microsecond, func(dbf *datastructure.DataBuffer) {
		if ac.clientConn != nil {
			ac.clientConn.Write(dbf.Data[:dbf.Length], false)
		}
		ac.encodePool.PoolPut(dbf)
	})

	return ac
}

//Start start
func (ac *AppClientFec) Start() {
	go ac.socketToDevice()
	go ac.il.PushDown()
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
			if ac.duration > 0 {
				ac.il.Put(encodeResult)
			} else {
				for i := range encodeResult {
					item := encodeResult[i]
					if ac.clientConn != nil {
						ac.clientConn.Write(item.Data[:item.Length], false)
					}
					ac.encodePool.PoolPut(item)
				}
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
