package codec

import (
	"encoding/binary"
	"sync"
	"time"
)

type StageBuffer struct {
	buffer   []byte
	capacity int
	size     int
	cursor   int
	lock     sync.Mutex
	sTimer   *time.Timer
	waitTime time.Duration
}

func NewStageBuffer(cap int) *StageBuffer {
	sbuffer := new(StageBuffer)
	sbuffer.capacity = cap
	sbuffer.buffer = make([]byte, 2000*cap)
	sbuffer.size = 0
	sbuffer.cursor = 0
	sbuffer.waitTime = time.Duration(20) * time.Millisecond //todo 配置
	return sbuffer
}

func (sb *StageBuffer) Append(data []byte, length uint16, resultBuffer []byte, codec *FecCodec, callback func(*StageBuffer, []byte, int)) {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	if sb.size == 0 {
		sb.sTimer = time.NewTimer(sb.waitTime)
		go func() {
			<-sb.sTimer.C
			sb.lock.Lock()
			defer sb.lock.Unlock()
			alignLen := codec.Align(sb.cursor)
			copy(resultBuffer, sb.buffer[:sb.cursor])
			callback(sb, resultBuffer[:alignLen], sb.cursor)
			sb.cursor = 0
			sb.size = 0
		}()
	}

	binary.BigEndian.PutUint16(sb.buffer[sb.cursor:], length)
	sb.cursor = sb.cursor + 2
	copy(sb.buffer[sb.cursor:], data)
	sb.cursor = sb.cursor + int(length)
	sb.size = sb.size + 1
	if sb.size == sb.capacity {
		sb.sTimer.Stop()
		alignLen := codec.Align(sb.cursor)
		copy(resultBuffer, sb.buffer[:sb.cursor])
		callback(sb, resultBuffer[:alignLen], sb.cursor)
		sb.cursor = 0
		sb.size = 0
	}
}
