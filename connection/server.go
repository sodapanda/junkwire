package connection

import (
	"fmt"
	"net"

	"github.com/google/netstack/tcpip"
	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/device"
)

//ServerConnHandler handler callback
type ServerConnHandler interface {
	OnData([]byte, *ServerConn)
	OnDisconnect()
}

//ServerConn server connection
type ServerConn struct {
	tun                 *device.TunInterface
	srcIP               tcpip.Address
	dstIP               tcpip.Address
	srcPort             uint16
	dstPort             uint16
	payloadsFromUpLayer *ds.BlockingQueue
	lastRcvSeq          uint32
	lastRcvAck          uint32
	lastRcvLen          uint32
	fsm                 *ds.Fsm
	sendID              uint16
	handler             ServerConnHandler
	pool                *ds.DataBufferPool
}

//NewServerConn create server connection
func NewServerConn(srcIP string, srcPort uint16, tun *device.TunInterface) *ServerConn {
	sc := new(ServerConn)
	sc.tun = tun
	sc.srcIP = tcpip.Address(net.ParseIP(srcIP).To4())
	sc.srcPort = srcPort
	sc.payloadsFromUpLayer = ds.NewBlockingQueue(500)
	sc.pool = ds.NewDataBufferPool()

	sc.fsm = ds.NewFsm("stop")

	sc.fsm.AddRule("stop", ds.Event{Name: "start"}, "waitsyn", func(et ds.Event) {
		fmt.Println("server wait syn")
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvsyn"}, "gotSyn", func(et ds.Event) {
		fmt.Println("server got syn then send syn ack")
		cp := et.ConnPacket.(ConnPacket)
		sc.dstIP = cp.srcIP
		sc.dstPort = cp.srcPort

		cp = ConnPacket{}
		cp.syn = true
		cp.ack = true
		cp.srcIP = sc.srcIP
		cp.dstIP = sc.dstIP
		cp.srcPort = sc.srcPort
		cp.dstPort = sc.dstPort
		cp.seqNum = sc.lastRcvAck
		cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
		cp.payload = nil
		result := make([]byte, 40)
		len := cp.encode(result)

		if len > 40 {
			fmt.Println("send syn ack wrong")
		}

		sc.tun.Write(result)
		sc.sendID++
		sc.fsm.OnEvent(ds.Event{Name: "sdsynack"})
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvack"}, "error", func(et ds.Event) {
		fmt.Println("wait syn while error :got ack")
		sc.reset()
		sc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvrst"}, "waitsyn", func(et ds.Event) {
		fmt.Println("wait syn got rst.Stay")
	})

	sc.fsm.AddRule("gotSyn", ds.Event{Name: "sdsynack"}, "synacksd", func(et ds.Event) {
		fmt.Println("syn ack sent")
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvsyn"}, "error", func(et ds.Event) {
		fmt.Println("synacksd rcvsyn error")
		sc.reset()
		sc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvack"}, "estb", func(et ds.Event) {
		fmt.Println("server estab")
		cp := et.ConnPacket.(ConnPacket)
		if cp.payload != nil && len(cp.payload) > 0 {
			sc.handler.OnData(cp.payload, sc)
		}
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvrst"}, "error", func(et ds.Event) {
		fmt.Println("synacksd rcvrst error")
		sc.reset()
		sc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvsyn"}, "error", func(et ds.Event) {
		fmt.Println("estb rcvsyn error")
		sc.reset()
		sc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvack"}, "estb", func(et ds.Event) {
		cp := et.ConnPacket.(ConnPacket)
		if cp.payload != nil && len(cp.payload) > 0 {
			sc.handler.OnData(cp.payload, sc)
		}
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvrst"}, "error", func(et ds.Event) {
		fmt.Println("estb rcvrst error")
		sc.reset()
		sc.fsm.OnEvent(ds.Event{Name: "sdrst"})
	})

	sc.fsm.AddRule("error", ds.Event{Name: "sdrst"}, "waitsyn", func(et ds.Event) {
		fmt.Println("return to wait syn")
	})

	sc.fsm.OnEvent(ds.Event{Name: "start"})

	go sc.q2Tun()
	go sc.readLoop()
	return sc
}

//AddHandler add handler callback
func (sc *ServerConn) AddHandler(handler ServerConnHandler) {
	sc.handler = handler
}

func (sc *ServerConn) readLoop() {
	for {
		dataBuffer := sc.tun.Read()
		cp := ConnPacket{}
		if dataBuffer == nil || dataBuffer.Length == 0 {
			fmt.Println("server conn loop exit")
			return
		}
		cp.decode(dataBuffer.Data[:dataBuffer.Length])
		sc.lastRcvSeq = cp.seqNum
		sc.lastRcvAck = cp.ackNum
		sc.lastRcvLen = uint32(len(cp.payload))
		if cp.syn {
			sc.lastRcvLen = 1
		}

		if cp.push {
			sc.Write(cp.payload, true)
			sc.tun.Recycle(dataBuffer)
			continue
		}
		et := ds.Event{}
		if cp.syn {
			et.Name = "rcvsyn"
		}
		if cp.ack {
			et.Name = "rcvack"
		}
		if cp.rst {
			et.Name = "rcvrst"
		}
		et.ConnPacket = cp
		sc.fsm.OnEvent(et)
		sc.tun.Recycle(dataBuffer)
	}
}

func (sc *ServerConn) reset() {
	fmt.Println("send reset")
	cp := ConnPacket{}
	cp.syn = false
	cp.ack = false
	cp.rst = true
	cp.srcIP = sc.srcIP
	cp.dstIP = sc.dstIP
	cp.srcPort = sc.srcPort
	cp.dstPort = sc.dstPort
	cp.seqNum = sc.lastRcvAck
	cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
	cp.payload = nil
	result := make([]byte, 40)
	cp.encode(result)
	sc.tun.Write(result)
	sc.sendID = 0
}

func (sc *ServerConn) Write(data []byte, isKp bool) {
	dbf := sc.pool.PoolGet()
	cp := ConnPacket{}
	cp.ipID = sc.sendID
	sc.sendID++
	cp.srcIP = sc.srcIP
	cp.dstIP = sc.dstIP
	cp.srcPort = sc.srcPort
	cp.dstPort = sc.dstPort
	cp.syn = false
	cp.ack = true
	cp.rst = false
	if isKp {
		cp.push = true
	}
	cp.seqNum = sc.lastRcvAck
	cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
	cp.payload = data
	length := cp.encode(dbf.Data)
	dbf.Length = int(length)
	sc.payloadsFromUpLayer.Put(dbf)
}

func (sc *ServerConn) q2Tun() {
	for {
		dbf := sc.payloadsFromUpLayer.Get()
		sc.tun.Write(dbf.Data[:dbf.Length])
		sc.pool.PoolPut(dbf)
	}
}
