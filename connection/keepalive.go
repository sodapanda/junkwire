package connection

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

type keeper struct {
	callback   func()
	cc         *ClientConn
	kpMap      map[uint64]uint64
	dataBuffer []byte
	lock       *sync.Mutex
	stopFlag   bool
}

func newKeeper(cc *ClientConn, lostCallback func()) *keeper {
	kp := new(keeper)
	kp.callback = lostCallback
	kp.dataBuffer = make([]byte, 8)
	kp.lock = new(sync.Mutex)
	kp.cc = cc
	kp.stopFlag = false
	return kp
}

func (kp *keeper) start() {
	for {
		kp.kpMap = make(map[uint64]uint64)
		for i := 0; i < 5; i++ {
			time.Sleep(100 * time.Millisecond)
			kp.send()
		}

		time.Sleep(2 * time.Second)

		kp.lock.Lock()
		lossCount := 0
		for _, rtt := range kp.kpMap {
			if rtt == 0 {
				lossCount++
			}
		}
		kp.lock.Unlock()
		if lossCount > 3 {
			fmt.Println("disconnect! keep alive loss ", lossCount)
			kp.callback()
			break
		}
		if kp.stopFlag {
			break
		}
	}
}

func (kp *keeper) send() {
	kp.lock.Lock()
	defer kp.lock.Unlock()
	unixNano := time.Now().UnixNano()
	binary.BigEndian.PutUint64(kp.dataBuffer, uint64(unixNano))
	kp.cc.Write(kp.dataBuffer, true)
	kp.kpMap[uint64(unixNano)] = 0
}

func (kp *keeper) rcv(timeStamp uint64) {
	kp.lock.Lock()
	kp.lock.Unlock()
	unixNano := time.Now().UnixNano()
	rtt := uint64(unixNano) - timeStamp
	if _, ok := kp.kpMap[timeStamp]; ok {
		kp.kpMap[timeStamp] = rtt
	} else {
		fmt.Println("kp map has no key ", timeStamp)
	}
}

func (kp *keeper) stop() {
	kp.stopFlag = true
}
