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
	tm           *time.Timer
	lock         *sync.Mutex
	callback     func(*StageBuffer, []byte, int)
	icodec       *FecCodec
	fullCh       chan bool
	waitDoneCh   chan bool
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
	sbuffer.callback = callback
	sbuffer.icodec = icodec
	sbuffer.fullCh = make(chan bool)
	sbuffer.waitDoneCh = make(chan bool)
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
		go func() { //todo leak
			select {
			case <-sb.tm.C:
				sb.lock.Lock()
				if sb.size == 0 {
					//空 不操作
					fmt.Println("go2 wake up timer size 0")
					sb.waitDoneCh <- true
				}
				if sb.size > 0 && sb.size < sb.capacity {
					//超时没满 发送数据 不用通知
					sb.sendOut()
				}

				if sb.size == sb.capacity {
					//超时 满了:不发送数据 通知
					sb.waitDoneCh <- true
				}
				sb.lock.Unlock()
			case <-sb.fullCh:
				//没超时
				sb.waitDoneCh <- true
			}
		}()
		sb.lock.Unlock()
	} else if sb.size == sb.capacity { //如果满了 发出去
		sb.sendOut()
		sb.lock.Unlock()
		expr := !sb.tm.Stop() //true:调用stop的时候还没到期 false:调用stop的时候已经到期了 需要检查chann有没有关闭
		if expr {
			fmt.Println("full,go2 waiting")
			<-sb.waitDoneCh
		} else {
			sb.fullCh <- true
			<-sb.waitDoneCh
		}
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
