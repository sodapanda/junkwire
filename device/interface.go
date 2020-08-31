package device

import (
	"os/exec"
	"time"

	ds "github.com/sodapanda/junkwire/datastructure"
	"github.com/sodapanda/junkwire/misc"
	"github.com/songgao/water"
)

//TunInterface tun device
type TunInterface struct {
	name  string
	tun   *water.Interface
	queue *ds.BlockingQueue
	pool  *ds.DataBufferPool
}

//NewTunInterface 创建tun设备
func NewTunInterface(interfaceName string, address string, queueLen int) *TunInterface {
	d := new(TunInterface)
	conf := water.Config{
		DeviceType: water.TUN,
	}
	conf.Name = interfaceName
	d.tun, _ = water.New(conf)

	cmd := exec.Command("sudo", "ip", "address", "add", address+"/24", "dev", interfaceName)
	cmd.Run()
	cmd = exec.Command("sudo", "ip", "link", "set", "up", "dev", interfaceName)
	cmd.Run()

	d.queue = ds.NewBlockingQueue(queueLen)
	d.pool = ds.NewDataBufferPool()

	go d.turnUp()
	return d
}

func (d *TunInterface) turnUp() {
	for {
		dbf := d.pool.PoolGet()
		length, err := d.tun.Read(dbf.Data)
		misc.CheckErr(err)
		dbf.Length = length
		d.queue.Put(dbf)
	}
}

func (d *TunInterface) Read() *ds.DataBuffer {
	dbf := d.queue.Get()
	return dbf
}

func (d *TunInterface) Interrupt() {
	d.queue.Interrupt()
}

func (d *TunInterface) Recycle(dbf *ds.DataBuffer) {
	d.pool.PoolPut(dbf)
}

func (d *TunInterface) Write(data []byte) (int, error) {
	return d.tun.Write(data)
}

func (d *TunInterface) ReadTimeout(timeout time.Duration) *ds.DataBuffer {
	dbf := d.queue.GetWithTimeout(timeout)
	return dbf
}

func (d *TunInterface) ClearQueue() {
	for d.queue.GetSize() != 0 {
		dbf := d.queue.Get()
		d.pool.PoolPut(dbf)
	}
}
