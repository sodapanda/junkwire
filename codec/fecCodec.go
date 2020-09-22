package codec

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/klauspost/reedsolomon"
	"github.com/sodapanda/junkwire/datastructure"
)

//FecCodec fec
type FecCodec struct {
	segCount            int
	fecSegCount         int
	encodeWorkspace     [][]byte
	decodeLinkMap       map[uint64][]*FtPacket
	keyList             *list.List
	decodeMapCapacity   int
	decodeTempWorkspace [][]byte
	encoder             reedsolomon.Encoder
	tmpPool             [][]byte //因为每次fec桶大小不一样
	currentID           uint64
	fullPacketHolder    []byte //分包合并然后拆分用的内存空间
	lenKindMap          map[int]int
}

//NewFecCodec new
func NewFecCodec(segCount int, fecSegCount int, decodeMapCap int) *FecCodec {
	codec := new(FecCodec)
	codec.segCount = segCount
	codec.fecSegCount = fecSegCount
	codec.encodeWorkspace = make([][]byte, segCount+fecSegCount)
	codec.decodeLinkMap = make(map[uint64][]*FtPacket)
	codec.lenKindMap = make(map[int]int)
	codec.keyList = list.New()
	codec.decodeMapCapacity = decodeMapCap
	codec.decodeTempWorkspace = make([][]byte, segCount+fecSegCount)
	codec.encoder, _ = reedsolomon.New(segCount, fecSegCount)
	codec.tmpPool = make([][]byte, fecSegCount)
	for i := range codec.tmpPool {
		codec.tmpPool[i] = make([]byte, 2000)
	}

	codec.fullPacketHolder = make([]byte, 2000*(segCount+fecSegCount))

	return codec
}

//Encode encode
func (codec *FecCodec) Encode(data []byte, realLength int, result []*datastructure.DataBuffer) {
	segSize := (len(data)) / codec.segCount
	for i := 0; i < codec.segCount; i++ {
		start := i * segSize
		end := start + segSize
		codec.encodeWorkspace[i] = data[start:end]
	}

	for i := 0; i < codec.fecSegCount; i++ {
		codec.encodeWorkspace[codec.segCount+i] = codec.tmpPool[i][:segSize]
	}

	codec.encoder.Encode(codec.encodeWorkspace)

	codec.currentID = codec.currentID + 1

	for i, data := range codec.encodeWorkspace {
		ftp := new(FtPacket)
		ftp.gID = codec.currentID
		ftp.index = uint16(i)
		ftp.realLength = uint16(realLength)
		ftp.data = data
		codeLen := ftp.Encode(result[i].Data)
		result[i].Length = codeLen //指示有效长度
	}
}

//Decode decode
func (codec *FecCodec) Decode(ftp *FtPacket, result []*datastructure.DataBuffer) bool {
	_, found := codec.decodeLinkMap[ftp.gID]
	if !found {
		codec.decodeLinkMap[ftp.gID] = make([]*FtPacket, codec.segCount+codec.fecSegCount)
		codec.keyList.PushBack(ftp.gID)
	}

	if len(codec.decodeLinkMap) > codec.decodeMapCapacity {
		firstKeyElm := codec.keyList.Front()
		firstKey := firstKeyElm.Value.(uint64)
		ftps := codec.decodeLinkMap[firstKey]
		for _, ftp := range ftps {
			if ftp != nil {
				mFtPool.poolPut(ftp)
			}
		}
		delete(codec.decodeLinkMap, firstKey)
		codec.keyList.Remove(firstKeyElm)
	}

	poolFtp := mFtPool.poolGet()
	if poolFtp == nil {
		fmt.Println("poolFtp is nil!!")
	}
	poolFtp.len = len(ftp.data)
	poolFtp.gID = ftp.gID
	poolFtp.index = ftp.index
	poolFtp.realLength = ftp.realLength
	copy(poolFtp.data, ftp.data)

	row := codec.decodeLinkMap[ftp.gID]
	if row[ftp.index] != nil {
		fmt.Println("Dup!", ftp.gID, ftp.index)
	}
	row[ftp.index] = poolFtp

	gotCount := 0
	allSegGot := false
	for i, v := range row {
		if v != nil {
			gotCount++
		}
		if i == codec.segCount-1 && gotCount == codec.segCount {
			allSegGot = true
		}
	}

	if gotCount != codec.segCount {
		return false
	}

	for i := range row {
		thisFtp := row[i]
		if thisFtp != nil {
			codec.decodeTempWorkspace[i] = thisFtp.data[:thisFtp.len]
		} else {
			codec.decodeTempWorkspace[i] = make([]byte, 0, len(ftp.data))
		}
	}

	if !allSegGot {
		codec.encoder.Reconstruct(codec.decodeTempWorkspace)

		//
		ftpDataLen := len(ftp.data)
		_, lenKindFound := codec.lenKindMap[ftpDataLen]
		if !lenKindFound {
			codec.lenKindMap[ftpDataLen] = 1
		} else {
			codec.lenKindMap[ftpDataLen] = codec.lenKindMap[ftpDataLen] + 1
		}
		//
	}

	fCursor := 0
	for i, data := range codec.decodeTempWorkspace {
		if i == codec.segCount {
			break
		}
		copy(codec.fullPacketHolder[fCursor:], data)
		fCursor = fCursor + len(data)
	}

	fullData := codec.fullPacketHolder[:ftp.realLength]

	sCursor := 0
	//如果是超时过来的话，只有一个包
	for i := 0; i < codec.segCount; i++ {
		if sCursor >= len(fullData) {
			break
		}
		length := binary.BigEndian.Uint16(fullData[sCursor:])
		sCursor = sCursor + 2
		copy(result[i].Data, fullData[sCursor:sCursor+int(length)])
		sCursor = sCursor + int(length)
		result[i].Length = int(length)
	}

	return true
}

//Align align
func (codec *FecCodec) Align(length int) int {
	minBucket := math.Ceil(float64(length) / float64(codec.segCount))
	return int(minBucket) * codec.segCount
}

//Dump dump
func (codec *FecCodec) Dump() string {
	var sb strings.Builder
	inCompCount := 0
	for e := codec.keyList.Front(); e != nil; e = e.Next() {
		gotCount := 0
		fKey := e.Value.(uint64)
		ftPkts := codec.decodeLinkMap[fKey]
		fmt.Fprintf(&sb, "%d", fKey)
		for _, pkt := range ftPkts {
			if pkt == nil {
				fmt.Fprintf(&sb, "❌")
			} else {
				fmt.Fprintf(&sb, "✅")
				gotCount++
			}
		}
		if gotCount < codec.segCount {
			inCompCount++
		}
		fmt.Fprintf(&sb, "\n")
	}
	fmt.Fprintf(&sb, "not complete row %d", inCompCount)
	return sb.String()
}

//DumpLenKind lenkind
func (codec *FecCodec) DumpLenKind() string {
	var sb strings.Builder
	for lenKind, count := range codec.lenKindMap {
		fmt.Fprintf(&sb, "%d:%d\n", lenKind, count)
	}
	fmt.Fprintf(&sb, "total kind:%d\n", len(codec.lenKindMap))
	return sb.String()
}
