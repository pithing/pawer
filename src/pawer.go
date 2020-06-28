package main

import (
	"container/list"
	"net"
	"sync"
	"time"
)

var Config = ConfDefault()
var Way = OneWay{
	Version: Config.Version,
	Receive: &PackageQueue{
		Data: &list.List{},
		Lock: sync.Mutex{},
	},
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
	go Way.WayConnIO()
	for true {
		packets := Way.Receive.Pop()
		for _, packet := range packets {
			OnWayReceive(packet)
		}
		if len(packets) == 0 {
			time.Sleep(time.Second / 100)
		}
	}
}
