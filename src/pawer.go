package main

import (
	"net"
)

var Config = ConfDefault()
var Way = OneWay{
	Version: Config.Version,
}

func main() {
	Way.Default(Config.Local, Config.Remote)
	//开启本地监听服务
	for _, link := range Config.Link {
		local, err := net.ResolveTCPAddr("tcp", link.Local)
		if err != nil {
			panic(err)
		}
		remote, err := net.ResolveTCPAddr("tcp", link.Remote)
		if err != nil {
			panic(err)
		}
		go Transfer(local, remote)
	}
	//开启心跳
	go BreakHeart()
	//来自单向的数据包
	Way.ReceiveAction = func(packet *Package) {
		OnWayReceive(packet)
	}
	Way.WayConnIO()
}
