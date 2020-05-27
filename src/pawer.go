package main

import "net"

var Way = OneWay{
	Version: Config.Version,
}.Default(Config.Local, Config.Remote)

func main() {
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
	for {
		select {
		case packet := <-Way.Reader:
			OnWayReceive(packet)
		}
	}
}
