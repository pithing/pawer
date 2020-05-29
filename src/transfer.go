package main

import (
	"net"
	"os"
	"time"
)

var Timer = time.Now().Unix()

var Clients = make(map[string]chan *Package)
var Remotes = make(map[string]chan *Package)
var BufferSize = 1024 * 512

func ChanIO(source map[string]chan *Package, local *net.TCPAddr, remote *net.TCPAddr, packet *Package) {
	disconnect := func() {
		Way.Sender <- &Package{
			Type:   0xff,
			Data:   []byte{},
			Local:  local,
			Remote: remote,
		}
	}
	radix := func(client *net.TCPConn) {
		buffer := make([]byte, BufferSize)
		for {
			count, err := client.Read(buffer)
			if err != nil {
				disconnect()
				break
			}
			Way.Sender <- &Package{
				Type:   1,
				Data:   buffer[:count],
				Local:  local,
				Remote: remote,
			}
		}
	}
	var user = local.String()
	var client *net.TCPConn
	var err error
	for {
		item := source[user]
		if item == nil {
			source[user] = make(chan *Package)
			item = source[user]
			if packet != nil {
				item <- packet
			}
			client, err = net.DialTCP("tcp", nil, remote)
			if err != nil {
				disconnect()
				break
			}
			_ = client.SetKeepAlive(true)
			go radix(client)
		}
		select {
		case packet := <-item:
			_, err = client.Write(packet.Data)
			if err != nil {
				disconnect()
				break
			}
			continue
		}
		break
	}
}

func BreakHeart() {
	zero, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	for {
		time.Sleep(time.Second * 28)
		Way.Sender <- &Package{
			Type:   0xC0,
			Data:   []byte{},
			Local:  zero,
			Remote: zero,
		}
		//判断心跳
		if time.Now().Unix()-Timer >= 60 {
			Way.Sender <- &Package{
				Type:   0xF0,
				Data:   []byte{},
				Local:  zero,
				Remote: zero,
			}
			time.Sleep(time.Second * 5)
			_ = (&os.Process{Pid: os.Getpid()}).Kill()
		}
	}
}

func OnWayReceive(packet *Package) {
	user := packet.Local.String()
	switch packet.Type {
	case 0: //请求
		remote, has := Remotes[user]
		if !has {
			go ChanIO(Remotes, packet.Local, packet.Remote, packet)
		} else {
			remote <- packet
		}
		break
	case 1: //响应
		item, has := Clients[user]
		if has {
			item <- packet
		}
		break
	case 0xC0: //心跳
		Timer = time.Now().Unix()
		break
	case 0xF0: //结束
		_ = (&os.Process{Pid: os.Getpid()}).Kill()
		break
	case 0xFF: //断开
		delete(Clients, user)
		break
	}
}
func Transfer(local *net.TCPAddr, remote *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", local)
	if err != nil {
		panic(err)
	}
	for {
		client, err := listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		_ = client.SetKeepAlive(true)
		go ClientIO(client, remote)
	}
}

func ClientIO(client *net.TCPConn, remote *net.TCPAddr) {
	user := client.RemoteAddr().String()
	local, _ := net.ResolveTCPAddr("tcp", user)
	go ChanIO(Clients, local, remote, nil)
	buffer := make([]byte, BufferSize)
	for {
		count, err := client.Read(buffer)
		if err != nil {
			Way.Sender <- &Package{
				Type:   0xff,
				Data:   []byte{},
				Local:  local,
				Remote: remote,
			}
			break
		}
		Way.Sender <- &Package{
			Type:   0,
			Data:   buffer[:count],
			Local:  local,
			Remote: remote,
		}
	}
}
