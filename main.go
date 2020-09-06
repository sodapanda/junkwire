package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/sodapanda/junkwire/application"
	"github.com/sodapanda/junkwire/connection"
	"github.com/sodapanda/junkwire/device"
	"github.com/sodapanda/junkwire/misc"
)

func main() {
	fmt.Println("start")

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

func client(config *Config) {
	tun := device.NewTunInterface("faketcp", config.Client.Tun.DeviceIP, 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	client := application.NewAppClient(config.Client.Socket.ListenPort)
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
		fmt.Println("client stop restart")
		time.Sleep(5 * time.Second)
	}
}

func server(config *Config) {
	tun := device.NewTunInterface("faketcp", config.Server.Tun.DeviceIP, 100)

	fmt.Println("continue?")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	serPort, _ := strconv.Atoi(config.Server.Tun.Port)

	sc := connection.NewServerConn(config.Server.Tun.SrcIP, uint16(serPort), tun)
	sv := application.NewAppServer(config.Server.Socket.DstIP, config.Server.Socket.DstPort, sc)
	sv.Start()
	reader = bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println(sc)
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
}
