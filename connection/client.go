package connection

import (
	"fmt"
	"net"

	"github.com/google/netstack/tcpip"
	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/device"
)

//ClientConnHandler handler
type ClientConnHandler interface {
	OnData([]byte)
	OnDisconnect()
}

//ClientConn client connection
type ClientConn struct {
	tun                 *device.TunInterface
	srcIP               tcpip.Address
	dstIP               tcpip.Address
	srcPort             uint16
	dstPort             uint16
	sendSeq             uint32
	lastRcvSeq          uint32
	payloadsFromUpLayer *ds.BlockingQueue
	pool                *ds.DataBufferPool
	fsm                 *ds.Fsm
	handler             ClientConnHandler
}

//NewClientConn new client connection
func NewClientConn(tun *device.TunInterface, srcIP string, dstIP string, srcPort uint16, dstPort uint16, handler ClientConnHandler) *ClientConn {
	cc := new(ClientConn)
	cc.pool = ds.NewDataBufferPool()
	cc.payloadsFromUpLayer = ds.NewBlockingQueue(500)
	cc.tun = tun
	cc.srcIP = tcpip.Address(net.ParseIP(srcIP).To4())
	cc.dstIP = tcpip.Address(net.ParseIP(dstIP).To4())
	cc.srcPort = srcPort
	cc.dstPort = dstPort
	cc.sendSeq = 1000
	cc.handler = handler

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
		cp.ackNum = 0
		cp.seqNum = cc.sendSeq
		cp.payload = nil
		cp.rst = false

		result := make([]byte, 40)
		cp.encode(result)
		cc.tun.Write(result)
		cc.sendSeq++
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
		cp.ackNum = cc.lastRcvSeq
		cp.seqNum = cc.sendSeq
		cp.payload = nil
		cp.rst = false

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
	})

	cc.fsm.AddRule("estb", ds.Event{Name: "rcvsynack"}, "error", func(ev ds.Event) {
		fmt.Println("estb rcvsynack error")
		cc.reset()
		cc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	cc.fsm.AddRule("estb", ds.Event{Name: "rcvack"}, "estb", func(ev ds.Event) {
		fmt.Println("recv ack")
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
		cc.handler.OnDisconnect()
		fmt.Println("stop state")
	})

	cc.fsm.OnEvent(ds.Event{Name: "sdsyn"})

	go cc.readLoop()
	go cc.q2Tun()
	return cc
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
	cp.seqNum = cc.sendSeq
	cp.ackNum = cc.lastRcvSeq + 1
	cp.payload = nil
	result := make([]byte, 40)
	cp.encode(result)
	cc.tun.Write(result)
	cc.sendSeq = 1000
}

func (cc *ClientConn) readLoop() {
	for {
		dataBuffer := cc.tun.Read()
		cp := ConnPacket{}
		if dataBuffer == nil || dataBuffer.Length == 0 {
			fmt.Println("client conn loop exit")
			return
		}
		cp.decode(dataBuffer.Data[:dataBuffer.Length])
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
}

func (cc *ClientConn) Write(data []byte) {
	dbf := cc.pool.PoolGet()
	copy(dbf.Data, data)
	dbf.Length = len(data)
	cc.payloadsFromUpLayer.Put(dbf)
	cc.sendSeq = cc.sendSeq + uint32(dbf.Length)
}

func (cc *ClientConn) q2Tun() {
	for {
		dbf := cc.payloadsFromUpLayer.Get()
		cc.tun.Write(dbf.Data[:dbf.Length])
		cc.pool.PoolPut(dbf)
	}
}
