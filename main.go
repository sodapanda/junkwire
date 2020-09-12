package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sodapanda/junkwire/application"
	"github.com/sodapanda/junkwire/codec"
	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/device"
	"github.com/sodapanda/junkwire/misc"
)

var mCodec *codec.FecCodec

func main() {
	go ctlServer()
	misc.Init()
	misc.PLog("start")

	fConfigPath := flag.String("c", "config.json", "config file path")
	flag.Parse()
	configPath := *fConfigPath

	configFile, err := os.Open(configPath)
	misc.CheckErr(err)
	defer configFile.Close()
	configByte, _ := ioutil.ReadAll(configFile)
	mConfig := new(Config)
	json.Unmarshal(configByte, mConfig)

	isServer := mConfig.Mode == "server"

	if isServer {
		server(mConfig)
	} else {
		client(mConfig)
	}
}

func ctlServer() {
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, mCodec.Dump())
	})
	http.ListenAndServe(":8080", nil)
}

func client(config *Config) {
	tun := device.NewTunInterface("faketcp", config.Client.Tun.DeviceIP, 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	fmt.Println("start,please see log")

	var client application.IClient
	if config.Fec.Enable {
		misc.PLog("fec enable")
		codec := codec.NewFecCodec(config.Fec.Seg, config.Fec.Parity, config.Fec.Cap)
		mCodec = codec
		client = application.NewAppClientFec(config.Client.Socket.ListenPort, config.Fec.Seg, config.Fec.Parity, codec, config.Fec.Duration)
	} else {
		client = application.NewAppClient(config.Client.Socket.ListenPort)
	}
	client.Start()
	srcPort, _ := strconv.Atoi(config.Client.Tun.Port)

	connTimes := -1
	for {
		connTimes++
		if connTimes >= len(config.Client.Tun.Peers) {
			connTimes = 0
		}
		serConf := config.Client.Tun.Peers[connTimes]
		client.SetClientConn(nil)
		serPort, _ := strconv.Atoi(serConf.Port)
		cc := connection.NewClientConn(tun, config.Client.Tun.SrcIP, serConf.IP, uint16(srcPort), uint16(serPort))
		client.SetClientConn(cc)
		cc.WaitStop()
		misc.PLog("client main loop stop restart")
		time.Sleep(1 * time.Second)
	}
}

func server(config *Config) {
	tun := device.NewTunInterface("faketcp", config.Server.Tun.DeviceIP, 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	fmt.Println("start,please see log")

	serPort, _ := strconv.Atoi(config.Server.Tun.Port)

	sc := connection.NewServerConn(config.Server.Tun.SrcIP, uint16(serPort), tun)

	var sv application.IServer
	if config.Fec.Enable {
		misc.PLog("fec enable")
		codec := codec.NewFecCodec(config.Fec.Seg, config.Fec.Parity, config.Fec.Cap)
		mCodec = codec
		sv = application.NewAppServerFec(config.Server.Socket.DstIP, config.Server.Socket.DstPort, sc, config.Fec.Seg, config.Fec.Parity, codec, config.Fec.Duration)
	} else {
		sv = application.NewAppServer(config.Server.Socket.DstIP, config.Server.Socket.DstPort, sc)
	}
	sv.Start()
	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

//Config config
type Config struct {
	Mode   string `json:"mode"`
	Server struct {
		Tun struct {
			DeviceIP string `json:"deviceIP"`
			Port     string `json:"port"`
			SrcIP    string `json:"srcIP"`
		} `json:"tun"`
		Socket struct {
			DstIP   string `json:"dstIP"`
			DstPort string `json:"dstPort"`
		} `json:"socket"`
	} `json:"server"`
	Client struct {
		Tun struct {
			DeviceIP string `json:"deviceIP"`
			Port     string `json:"port"`
			SrcIP    string `json:"srcIP"`
			Peers    []struct {
				IP   string `json:"ip"`
				Port string `json:"port"`
			} `json:"peers"`
		} `json:"tun"`
		Socket struct {
			ListenPort string `json:"listenPort"`
		} `json:"socket"`
	} `json:"client"`
	Fec struct {
		Enable   bool `json:"enable"`
		Seg      int  `json:"seg"`
		Parity   int  `json:"parity"`
		Duration int  `json:"duration"`
		Cap      int  `json:"cap"`
	} `json:"fec"`
}
