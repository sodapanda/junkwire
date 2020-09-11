package application

import (
	"fmt"
	"math"
	"net"
	"time"

	"github.com/sodapanda/junkwire/codec"
	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/misc"
)

//AppServerFec AppServerFec
type AppServerFec struct {
	conn         *net.UDPConn
	serverConn   *connection.ServerConn
	seg          int
	parity       int
	duration     int //交织时间段
	codec        *codec.FecCodec
	encodePool   *datastructure.DataBufferPool
	decodeResult []*datastructure.DataBuffer
}

//NewAppServerFec NewAppServerFec
func NewAppServerFec(dstIP string, dstPort string, serverConn *connection.ServerConn, seg int, parity int, codec *codec.FecCodec, duration int) *AppServerFec {
	as := new(AppServerFec)
	address, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", dstIP, dstPort))
	misc.CheckErr(err)
	conn, err := net.DialUDP("udp4", nil, address)
	misc.CheckErr(err)
	as.conn = conn
	as.serverConn = serverConn
	as.seg = seg
	as.parity = parity
	as.duration = duration
	as.encodePool = datastructure.NewDataBufferPool()
	as.decodeResult = make([]*datastructure.DataBuffer, seg)
	as.codec = codec
	for i := range as.decodeResult {
		as.decodeResult[i] = new(datastructure.DataBuffer)
		as.decodeResult[i].Data = make([]byte, 2000)
	}
	return as
}

//Start start
func (as *AppServerFec) Start() {
	go as.socketToDevice()
	as.serverConn.AddHandler(handlerFec{ser: as})
}

func (as *AppServerFec) socketToDevice() {
	readBuf := make([]byte, 2000)
	sb := codec.NewStageBuffer(as.seg)
	fullDataBuffer := make([]byte, 2000*as.seg)
	gapF := float64(as.duration) / float64(as.seg+as.parity)
	gap := int(math.Ceil(gapF))

	for {
		length, err := as.conn.Read(readBuf)
		misc.CheckErr(err)
		data := readBuf[:length]
		encodeResult := make([]*datastructure.DataBuffer, as.seg+as.parity)
		sb.Append(data, uint16(length), fullDataBuffer, as.codec, func(cSb *codec.StageBuffer, resultData []byte, realLength int) {
			for i := range encodeResult {
				encodeResult[i] = as.encodePool.PoolGet()
			}

			as.codec.Encode(resultData, realLength, encodeResult)

			if as.duration > 0 {
				for i, data := range encodeResult {
					timer := time.NewTimer(time.Duration(gap*i) * time.Millisecond)

					go func(packetData *datastructure.DataBuffer) {
						<-timer.C
						if as.serverConn != nil {
							as.serverConn.Write(packetData.Data[:packetData.Length], false)
						}
						as.encodePool.PoolPut(packetData)
					}(data)
				}
			} else {
				for i := range encodeResult {
					item := encodeResult[i]
					as.serverConn.Write(item.Data[:item.Length], false)
					as.encodePool.PoolPut(item)
				}
			}
		})
	}
}

type handlerFec struct {
	ser *AppServerFec
}

func (h handlerFec) OnData(data []byte, conn *connection.ServerConn) {
	rcvPkt := new(codec.FtPacket)
	rcvPkt.Decode(data)

	done := h.ser.codec.Decode(rcvPkt, h.ser.decodeResult)
	if !done {
		return
	}

	for _, d := range h.ser.decodeResult {
		if d.Length == 0 {
			continue
		}
		_, err := h.ser.conn.Write(d.Data[:d.Length])
		d.Length = 0 //设置为0 表示没有内容
		misc.CheckErr(err)
	}
}

func (h handlerFec) OnDisconnect() {

}
