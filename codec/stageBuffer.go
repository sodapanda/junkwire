package codec

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

//StageBuffer buffer
type StageBuffer struct {
	buffer       []byte
	capacity     int
	size         int
	cursor       int
	resultBuffer []byte
	waitTime     time.Duration
	fullChann    chan bool
	tm           *time.Timer
	lock         *sync.Mutex
	channClosed  bool
	callback     func(*StageBuffer, []byte, int)
	icodec       *FecCodec
	done         chan bool
}

//NewStageBuffer new
func NewStageBuffer(icodec *FecCodec, cap int, rb []byte, timeout time.Duration, callback func(*StageBuffer, []byte, int)) *StageBuffer {
	sbuffer := new(StageBuffer)
	sbuffer.capacity = cap
	sbuffer.buffer = make([]byte, 2000*cap)
	sbuffer.size = 0
	sbuffer.cursor = 0
	sbuffer.resultBuffer = rb
	sbuffer.waitTime = timeout
	sbuffer.lock = new(sync.Mutex)
	sbuffer.fullChann = make(chan bool)
	sbuffer.done = make(chan bool)
	sbuffer.callback = callback
	sbuffer.icodec = icodec
	return sbuffer
}

var fullCount int

//Append append
func (sb *StageBuffer) Append(data []byte, length uint16) {
	//放入buffer
	sb.lock.Lock()
	binary.BigEndian.PutUint16(sb.buffer[sb.cursor:], length)
	sb.cursor = sb.cursor + 2
	copy(sb.buffer[sb.cursor:], data)
	sb.cursor = sb.cursor + int(length)
	sb.size = sb.size + 1

	if sb.size == 1 { //如果只有1个 开启timer
		if sb.tm == nil {
			sb.tm = time.NewTimer(sb.waitTime)
		} else {
			rstFlag := sb.tm.Reset(sb.waitTime)
			if rstFlag {
				fmt.Println("reset on not stop timer")
			}
		}
		sb.channClosed = false
		go func() {
			select {
			case <-sb.fullChann:
				fullCount++
				// fmt.Println("满了", fullCount)
				sb.lock.Lock()
				sb.sendOut()
				sb.lock.Unlock()
				sb.done <- true
			case <-sb.tm.C:
				sb.channClosed = true
				sb.lock.Lock()
				sb.sendOut()
				sb.lock.Unlock()
			}
		}()

		sb.lock.Unlock()
	} else if sb.size == sb.capacity { //如果满了 发出去
		sb.lock.Unlock()
		sb.fullChann <- true
		<-sb.done
		sb.lock.Lock()
		stopFlag := sb.tm.Stop() //true:调用stop的时候还没到期 false:调用stop的时候已经到期了 需要检查chann有没有关闭
		if !stopFlag {
			// fmt.Println("stop false")
			if !sb.channClosed {
				// fmt.Println("chann not closed when full")
				<-sb.tm.C
				sb.channClosed = true
			}
		}
		sb.lock.Unlock()
	} else {
		sb.lock.Unlock()
	}
}

func (sb *StageBuffer) sendOut() {
	alignLen := sb.icodec.Align(sb.cursor)
	copy(sb.resultBuffer, sb.buffer[:sb.cursor])
	sb.size = 0
	sb.callback(sb, sb.resultBuffer[:alignLen], sb.cursor)
	sb.cursor = 0
}
