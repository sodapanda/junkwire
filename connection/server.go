package connection

import (
	"fmt"
	"net"

	"github.com/google/netstack/tcpip"
	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/device"
	"github.com/sodapanda/junkwire/misc"
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
		misc.PLog("server wait syn")
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvsyn"}, "gotSyn", func(et ds.Event) {
		misc.PLog("server got syn then send syn ack")
		cp := et.ConnPacket.(ConnPacket)
		sc.dstIP = cp.srcIP
		sc.dstPort = cp.srcPort

		cp = ConnPacket{}
		cp.syn = true
		cp.ack = true
		cp.srcIP = sc.srcIP
		cp.dstIP = sc.dstIP
		cp.srcPort = sc.srcPort
		misc.PLog(fmt.Sprintf("    %s:%d", sc.dstIP.String(), sc.dstPort))
		cp.dstPort = sc.dstPort
		cp.seqNum = sc.lastRcvAck
		cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
		cp.payload = nil
		result := make([]byte, 40)
		cp.encode(result)
		sc.tun.Write(result)
		sc.sendID++
		sc.fsm.OnEvent(ds.Event{Name: "sdsynack"})
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvack"}, "waitsyn", func(et ds.Event) {
		misc.PLog("waitsyn rcvack. Stay")
	})

	sc.fsm.AddRule("waitsyn", ds.Event{Name: "rcvrst"}, "waitsyn", func(et ds.Event) {
		misc.PLog("wait syn got rst. Stay")
	})

	sc.fsm.AddRule("gotSyn", ds.Event{Name: "sdsynack"}, "synacksd", func(et ds.Event) {
		misc.PLog("syn ack sent")
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvsyn"}, "gotSyn", func(et ds.Event) {
		misc.PLog("\nsynacksd rcvsyn,new peer!")

		cp := et.ConnPacket.(ConnPacket)
		sc.dstIP = cp.srcIP
		sc.dstPort = cp.srcPort

		cp = ConnPacket{}
		cp.syn = true
		cp.ack = true
		cp.srcIP = sc.srcIP
		cp.dstIP = sc.dstIP
		cp.srcPort = sc.srcPort
		misc.PLog(fmt.Sprintf("    %s:%d", sc.dstIP.String(), sc.dstPort)) //上面被换过
		cp.dstPort = sc.dstPort
		cp.seqNum = sc.lastRcvAck
		cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
		cp.payload = nil
		result := make([]byte, 40)
		cp.encode(result)
		sc.tun.Write(result)
		sc.sendID = 0
		sc.fsm.OnEvent(ds.Event{Name: "sdsynack"})
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvack"}, "estb", func(et ds.Event) {
		misc.PLog("server estab")
		cp := et.ConnPacket.(ConnPacket)
		if cp.payload != nil && len(cp.payload) > 0 {
			sc.handler.OnData(cp.payload, sc)
		}
	})

	sc.fsm.AddRule("synacksd", ds.Event{Name: "rcvrst"}, "waitsyn", func(et ds.Event) {
		misc.PLog("synacksd rcvrst,to waitsyn")
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvsyn"}, "gotSyn", func(et ds.Event) {
		misc.PLog("\nestb rcvsyn,new peer!")
		cp := et.ConnPacket.(ConnPacket)
		sc.dstIP = cp.srcIP
		sc.dstPort = cp.srcPort

		cp = ConnPacket{}
		cp.syn = true
		cp.ack = true
		cp.srcIP = sc.srcIP
		cp.dstIP = sc.dstIP
		cp.srcPort = sc.srcPort
		misc.PLog(fmt.Sprintf("    %s:%d", sc.dstIP.String(), sc.dstPort))
		cp.dstPort = sc.dstPort
		cp.seqNum = sc.lastRcvAck
		cp.ackNum = sc.lastRcvSeq + sc.lastRcvLen
		cp.payload = nil
		result := make([]byte, 40)
		cp.encode(result)
		sc.tun.Write(result)
		sc.sendID = 0
		sc.fsm.OnEvent(ds.Event{Name: "sdsynack"})
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvack"}, "estb", func(et ds.Event) {
		cp := et.ConnPacket.(ConnPacket)
		if cp.payload != nil && len(cp.payload) > 0 {
			sc.handler.OnData(cp.payload, sc)
		}
	})

	sc.fsm.AddRule("estb", ds.Event{Name: "rcvrst"}, "waitsyn", func(et ds.Event) {
		misc.PLog("estb rcvrst,to waitsyn")
	})

	sc.fsm.AddRule("error", ds.Event{Name: "sdrst"}, "waitsyn", func(et ds.Event) {
		misc.PLog("return to wait syn")
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
			misc.PLog("server conn loop exit")
			return
		}
		cp.decode(dataBuffer.Data[:dataBuffer.Length])

		//不是syn包，并且不是当前peer的ip和port就丢掉
		if !cp.syn && cp.srcIP != sc.dstIP {
			misc.PLog("packet not from peer.drop")
			misc.PLog(fmt.Sprintf("    %s:%d", cp.srcIP.String(), cp.srcPort))
			sc.tun.Recycle(dataBuffer)
			continue
		}

		if cp.window != 6543 {
			misc.PLog("read window is not 6543!!Danger")
			misc.PLog(fmt.Sprintf("    %s:%d win:%d\n", cp.srcIP.String(), cp.srcPort, cp.window))
		}
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
	misc.PLog("send reset")
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
		data := dbf.Data[:dbf.Length]
		cp := ConnPacket{}
		cp.decode(data)
		if cp.dstIP != sc.dstIP {
			misc.PLog("write not to peer.Drop")
			sc.pool.PoolPut(dbf)
			continue
		}
		sc.tun.Write(data)
		sc.pool.PoolPut(dbf)
	}
}
