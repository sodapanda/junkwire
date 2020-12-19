JunkWire

功能：提高烂网的可用性。

UDP伪装TCP：伪装为TCP之后减少QoS

可配置多服务器，自动断线重连换服，上层无感知：心跳检测，每秒发送5个心跳包。如果5个心跳包都丢失的话就切换到下一个服务器。2秒即可完成。

可选FEC前向纠错：20个原始包加10个纠错包，就能将丢包率降低到万分之一。

交织编码：可以将短时间集中的丢包均匀开，防止原始包和纠错包都丢掉。

配置方法

junkwire使用tun设备进行数据传送，所以先确定运行环境支持tun设备。运行后会创建一个tun设备叫faketcp，这个设备ip地址可以通过配置文件配置。一般写成10.1.1.1即可，只要不跟自己本地网络环境冲突就行。运行后会虚拟一个网络设备10.1.1.2。

服务端配置

首先配置服务端DNAT，把对应端口的包转给junkwire处理。

首先开启ipv4转发，在 /etc/sysctl.conf 文件中添加 net.ipv4.ip_forward=1 然后运行sysctl -p使其生效。

添加iptables规则 iptables -t nat -A PREROUTING -i 网卡名 -d 网卡IP -p tcp --dport 17021(客户端连的端口) -j DNAT --to-destination 10.1.1.2:17021

wireguard配置举例，根据实际情况改动

```
    [Interface]
    Address = 10.200.201.1/24
    ListenPort = 21007
    #ListenPort = 12273
    PrivateKey = xxx
    MTU = 1340

    [Peer]
    PublicKey = xxx
    AllowedIPs = 10.200.201.2/32
    PersistentKeepalive = 25
```

服务端junkwire配置文件举例

```
    {
    "mode": "server",
    "queue":500,
    "server": {
        "tun": {
            "deviceIP": "10.1.1.1",
            "port": "17021",
            "srcIP": "10.1.1.2"
        },
        "socket": {
            "dstIP": "127.0.0.1",
            "dstPort": "21007"
        }
    },
    "fec": {
        "enable":false, //是否启用fec
        "seg": 20, //几个数据包
        "parity": 20, //几个纠错包
        "duration":0, //交织编码的时长
	    "stageTimeout":8, //桶没装满的话最长等多久
	    "cap":500, 
	    "row":1000
    }
    }
```

启动junkwire  ./junkwire -c 配置文件

启动wireguard wg-quick up wg0


客户端配置

客户端需要让虚拟设备10.1.1.2的数据顺利发送，需要snat

iptables -t nat -A POSTROUTING -s 10.1.1.2 -p tcp -o eth0 -j SNAT --to-source 本机网卡ip

路由配置，防止出口ip也被带进了wireguard

ip route add 服务端ip/32 via 本地网关 dev eth0

wireguard配置

```
[Interface]
Address = 10.200.201.2/24
PrivateKey = yJAu/oI+Oo/Mhswqbm3I/3PWYi+WSxX7JpTQ8IoQqWU=
MTU = 1340

[Peer]
PublicKey = 5/SgVv3hc3f5Fa/XoLo4isBzyrwwATs5sfQv7oWhiTM=
Endpoint = 127.0.0.1:21007
AllowedIPs = 0.0.0.0/1,128.0.0.0/1
PersistentKeepalive = 25
```

junkwire配置举例

```
{
    "mode": "client",
    "queue":500,
    "client": {
        "tun": {
            "deviceIP": "10.1.1.1",
            "port": "8978",
            "srcIP": "10.1.1.2",
            "peers": [
			{
			    "ip":"线路1",
			    "port":"50018"
			},
			{
			    "ip":"线路2",
			    "port":"17021"
			},
			{
			    "ip":"线路3",
			    "port":"17021"
			}
            ]
        },
        "socket": {
            "listenPort": "21007"
        }
    },
    "fec": {
        "enable":false,
        "seg": 20,
        "parity": 10,
        "stageTimeout": 8,
        "duration": 0,
        "cap": 500,
        "row": 1000
    }
}
```

启动junkwire ./junkwire -c 配置文件

启动wireguard wg-quick up wg0
