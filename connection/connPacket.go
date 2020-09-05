package connection

import (
	"github.com/google/netstack/tcpip"
	"github.com/google/netstack/tcpip/header"
	"github.com/google/netstack/tcpip/transport/tcp"
)

//ConnPacket connectiong packet IP TCP header
type ConnPacket struct {
	ipID    uint16
	srcIP   tcpip.Address
	dstIP   tcpip.Address
	srcPort uint16
	dstPort uint16
	syn     bool
	ack     bool
	rst     bool
	seqNum  uint32
	ackNum  uint32
	payload []byte
}

func (cp *ConnPacket) encode(result []byte) uint16 {
	copy(cp.payload, result[40:])
	ipPacket := header.IPv4(result[0:])
	//IP header
	ipHeader := header.IPv4Fields{}
	ipHeader.IHL = header.IPv4MinimumSize
	ipHeader.TOS = 0
	ipHeader.TotalLength = uint16(len(cp.payload) + 40)
	ipHeader.ID = cp.ipID
	ipHeader.Flags = 0b010
	ipHeader.FragmentOffset = 0
	ipHeader.TTL = 60
	ipHeader.Protocol = 6
	ipHeader.Checksum = 0
	ipHeader.SrcAddr = cp.srcIP.To4()
	ipHeader.DstAddr = cp.dstIP.To4()

	ipPacket.Encode(&ipHeader)
	ipPacket.SetChecksum(^ipPacket.CalculateChecksum())

	//TCP header
	tcpPacket := header.TCP(result[header.IPv4MinimumSize:])
	tcpHeader := header.TCPFields{}
	tcpHeader.SrcPort = cp.srcPort
	tcpHeader.DstPort = cp.dstPort
	tcpHeader.SeqNum = cp.seqNum
	tcpHeader.AckNum = cp.ackNum
	tcpHeader.DataOffset = header.TCPMinimumSize
	tcpHeader.Flags = 0
	if cp.syn {
		tcpHeader.Flags = tcpHeader.Flags | header.TCPFlagSyn
	}
	if cp.ack {
		tcpHeader.Flags = tcpHeader.Flags | header.TCPFlagAck
	}
	if cp.rst {
		tcpHeader.Flags = tcpHeader.Flags | header.TCPFlagRst
	}
	tcpHeader.WindowSize = 65000
	tcpHeader.Checksum = 0
	tcpHeader.UrgentPointer = 0

	tcpPacket.Encode(&tcpHeader)
	xsum := header.PseudoHeaderChecksum(tcp.ProtocolNumber, tcpip.Address(cp.srcIP), tcpip.Address(cp.dstIP), uint16(ipHeader.TotalLength-header.IPv4MinimumSize))
	xsum = header.Checksum(cp.payload, xsum)
	tcpPacket.SetChecksum(^tcpPacket.CalculateChecksum(xsum))

	return ipHeader.TotalLength
}

func (cp *ConnPacket) decode(data []byte) {
	ipHeader := header.IPv4(data[0:])
	tcpHeader := header.TCP(data[header.IPv4MinimumSize:])
	cp.ipID = ipHeader.ID()
	cp.srcIP = ipHeader.SourceAddress().To4()
	cp.dstIP = ipHeader.DestinationAddress().To4()
	cp.syn = tcpHeader.Flags()&header.TCPFlagSyn != 0
	cp.ack = tcpHeader.Flags()&header.TCPFlagAck != 0
	cp.rst = tcpHeader.Flags()&header.TCPFlagRst != 0
	cp.seqNum = tcpHeader.SequenceNumber()
	cp.ackNum = tcpHeader.AckNumber()
	cp.srcPort = tcpHeader.SourcePort()
	cp.dstPort = tcpHeader.DestinationPort()
	cp.payload = data[header.IPv4MinimumSize+header.TCPMinimumSize:] //todo 注意tcp mss的影响
}
