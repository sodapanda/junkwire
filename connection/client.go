package connection

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/netstack/tcpip"
	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/device"
)

//ClientConnHandler handler
type ClientConnHandler interface {
	OnData([]byte)
	OnDisconnect(cc *ClientConn)
	OnConnect(cc *ClientConn)
}

//ClientConn client connection
type ClientConn struct {
	tun                 *device.TunInterface
	srcIP               tcpip.Address
	dstIP               tcpip.Address
	srcPort             uint16
	dstPort             uint16
	sendID              uint16
	lastRcvSeq          uint32
	lastRcvLen          uint32
	lastRcvAck          uint32
	payloadsFromUpLayer *ds.BlockingQueue
	pool                *ds.DataBufferPool
	fsm                 *ds.Fsm
	handler             ClientConnHandler
	tunStopChan         chan string
	readLoopStopChan    chan string
	kp                  *keeper
}

//NewClientConn new client connection
func NewClientConn(tun *device.TunInterface, srcIP string, dstIP string, srcPort uint16, dstPort uint16) *ClientConn {
	cc := new(ClientConn)
	cc.pool = ds.NewDataBufferPool()
	cc.payloadsFromUpLayer = ds.NewBlockingQueue(500)
	cc.tun = tun
	cc.srcIP = tcpip.Address(net.ParseIP(srcIP).To4())
	cc.dstIP = tcpip.Address(net.ParseIP(dstIP).To4())
	cc.srcPort = srcPort
	cc.dstPort = dstPort
	cc.tunStopChan = make(chan string, 1)
	cc.readLoopStopChan = make(chan string, 1)
	cc.kp = newKeeper(cc, func() {
		cc.tun.Interrupt()
		cc.payloadsFromUpLayer.Interrupt()
		cc.handler.OnDisconnect(cc)
	})

	cc.fsm = ds.NewFsm("stop")

	cc.fsm.AddRule("stop", ds.Event{Name: "sdsyn"}, "synsd", func(ev ds.Event) {
		fmt.Println("send syn")
		cp := ConnPacket{}
		cp.syn = true
		cp.srcIP = cc.srcIP
		cp.dstIP = cc.dstIP
		cp.srcPort = cc.srcPort
		cp.dstPort = cc.dstPort
		cp.ack = false
		cp.ackNum = 1100
		cp.seqNum = 1000 //client的第一个seq是随机的
		cp.payload = nil
		cp.rst = false
		cp.ipID++

		result := make([]byte, 40)
		cp.encode(result)
		cc.tun.Write(result)
		go func() {
			time.Sleep(1 * time.Second)
			cc.fsm.OnEvent(ds.Event{Name: "synTimeout"})
		}()
	})

	cc.fsm.AddRule("synsd", ds.Event{Name: "synTimeout"}, "stop", func(ev ds.Event) {
		fmt.Println("timeout")

		cc.tun.Interrupt()
		cc.payloadsFromUpLayer.Interrupt()
		cc.handler.OnDisconnect(cc)
	})

	cc.fsm.AddRule("synsd", ds.Event{Name: "rcvsynack"}, "gotsynsck", func(ev ds.Event) {
		fmt.Println("got syn ack send ack")
		cp := ConnPacket{}
		cp.syn = false
		cp.srcIP = cc.srcIP
		cp.dstIP = cc.dstIP
		cp.srcPort = cc.srcPort
		cp.dstPort = cc.dstPort
		cp.ack = true
		cp.ackNum = cc.lastRcvSeq + cc.lastRcvLen
		cp.seqNum = cc.lastRcvAck
		cp.payload = nil
		cp.rst = false
		cp.ipID++

		result := make([]byte, 40)
		cp.encode(result)
		cc.tun.Write(result)
		cc.fsm.OnEvent(ds.Event{Name: "sdack"})
	})

	cc.fsm.AddRule("synsd", ds.Event{Name: "rcvrst"}, "error", func(ev ds.Event) {
		fmt.Println("synsd rcvrst error")
		cc.reset()
		cc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	cc.fsm.AddRule("gotsynsck", ds.Event{Name: "sdack"}, "estb", func(ev ds.Event) {
		fmt.Println("client estb")
		cc.handler.OnConnect(cc)
		go cc.kp.start()
	})

	cc.fsm.AddRule("estb", ds.Event{Name: "rcvsynack"}, "error", func(ev ds.Event) {
		fmt.Println("estb rcvsynack error")
		cc.reset()
		cc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	cc.fsm.AddRule("estb", ds.Event{Name: "rcvack"}, "estb", func(ev ds.Event) {
		cp := ev.ConnPacket.(ConnPacket)
		if cp.payload != nil && len(cp.payload) > 0 {
			cc.handler.OnData(cp.payload)
		}
	})

	cc.fsm.AddRule("estb", ds.Event{Name: "rcvrst"}, "error", func(ev ds.Event) {
		fmt.Println("estb reset")
		cc.reset()
		cc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	cc.fsm.AddRule("error", ds.Event{Name: "sdrst"}, "stop", func(ev ds.Event) {
		cc.tun.Interrupt()
		cc.payloadsFromUpLayer.Interrupt()
		cc.handler.OnDisconnect(cc)
		//todo 清理队列里没消费的
		fmt.Println("stop state")
	})

	cc.fsm.OnEvent(ds.Event{Name: "sdsyn"})

	go cc.readLoop(cc.readLoopStopChan)
	go cc.q2Tun(cc.tunStopChan)
	return cc
}

//AddHandler add callback
func (cc *ClientConn) AddHandler(handler ClientConnHandler) {
	cc.handler = handler
}

//WaitStop block wait for stop
func (cc *ClientConn) WaitStop() {
	<-cc.readLoopStopChan
	<-cc.tunStopChan
}

func (cc *ClientConn) reset() {
	fmt.Println("send reset")
	cp := ConnPacket{}
	cp.syn = false
	cp.ack = false
	cp.rst = true
	cp.srcIP = cc.srcIP
	cp.dstIP = cc.dstIP
	cp.srcPort = cc.srcPort
	cp.dstPort = cc.dstPort
	cp.seqNum = cc.lastRcvAck
	cp.ackNum = cc.lastRcvSeq + cc.lastRcvLen
	cp.payload = nil
	result := make([]byte, 40)
	cp.encode(result)
	cc.tun.Write(result)
	cc.sendID = 0
}

func (cc *ClientConn) readLoop(stopChan chan string) {
	for {
		dataBuffer := cc.tun.Read()
		cp := ConnPacket{}
		if dataBuffer == nil || dataBuffer.Length == 0 {
			fmt.Println("client conn loop exit")
			break
		}
		cp.decode(dataBuffer.Data[:dataBuffer.Length])
		cc.lastRcvSeq = cp.seqNum
		cc.lastRcvAck = cp.ackNum
		cc.lastRcvLen = uint32(len(cp.payload))
		if cp.syn {
			cc.lastRcvLen = 1
		}
		if cp.push { //心跳包处理
			content := binary.BigEndian.Uint64(cp.payload)
			cc.kp.rcv(content)
			cc.tun.Recycle(dataBuffer)
			continue
		}
		et := ds.Event{}
		if cp.syn && cp.ack {
			et.Name = "rcvsynack"
		} else if cp.ack {
			et.Name = "rcvack"
		}
		if cp.rst {
			et.Name = "rcvrst"
		}
		et.ConnPacket = cp
		cc.fsm.OnEvent(et)
		cc.tun.Recycle(dataBuffer)
	}

	stopChan <- "readLoopStop"
}

func (cc *ClientConn) Write(data []byte, isKp bool) {
	dbf := cc.pool.PoolGet()
	cp := ConnPacket{}
	cp.ipID = cc.sendID
	cc.sendID++
	cp.srcIP = cc.srcIP
	cp.dstIP = cc.dstIP
	cp.srcPort = cc.srcPort
	cp.dstPort = cc.dstPort
	cp.syn = false
	cp.ack = true
	cp.rst = false
	if isKp {
		cp.push = true
	}
	cp.seqNum = cc.lastRcvAck
	cp.ackNum = cc.lastRcvSeq + cc.lastRcvLen
	cp.payload = data
	length := cp.encode(dbf.Data)
	dbf.Length = int(length)
	cc.payloadsFromUpLayer.Put(dbf)
}

func (cc *ClientConn) q2Tun(stopChan chan string) {
	for {
		dbf := cc.payloadsFromUpLayer.Get()
		if dbf == nil {
			fmt.Println("q2tun read end")
			break
		}
		cc.tun.Write(dbf.Data[:dbf.Length])
		cc.pool.PoolPut(dbf)
	}

	stopChan <- "queue to tun stop"
}
