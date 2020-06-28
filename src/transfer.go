package main

import (
	"net"
	"os"
	"sync"
	"time"
)

var Timer = time.Now().Unix()

var Clients sync.Map
var Remotes sync.Map
var BufferSize = 5 * 1024 * 1024

func RemoteChanIO(local *net.TCPAddr, remote *net.TCPAddr, packet *Package) {
	disconnect := func() {
		Way.SendPacket(&Package{
			Type:   0xff,
			Data:   []byte{},
			Local:  local,
			Remote: remote,
		})
	}
	var user = local.String()
	var err error
	client, err := net.DialTCP("tcp", nil, remote)
	Remotes.Store(user, client)
	if err != nil {
		disconnect()
		return
	}
	_ = client.SetKeepAlive(true)
	if packet != nil {
		_, err = client.Write(packet.Data)
		if err != nil {
			disconnect()
			return
		}
	}
	buffer := make([]byte, BufferSize)
	for {
		if v, ok := Remotes.Load(user); ok {
			count, err := v.(*net.TCPConn).Read(buffer)
			if err != nil {
				disconnect()
				break
			}
			Way.SendPacket(&Package{
				Type:   1,
				Data:   buffer[:count],
				Local:  local,
				Remote: remote,
			})
		}
	}
}

func BreakHeart() {
	zero, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	for {
		time.Sleep(time.Second * 28)
		Way.SendPacket(&Package{
			Type:   0xC0,
			Data:   []byte{},
			Local:  zero,
			Remote: zero,
		})
		//判断心跳
		if time.Now().Unix()-Timer >= 60 {
			Way.SendPacket(&Package{
				Type:   0xF0,
				Data:   []byte{},
				Local:  zero,
				Remote: zero,
			})
			time.Sleep(time.Second * 5)
			_ = (&os.Process{Pid: os.Getpid()}).Kill()
		}
	}
}

func OnWayReceive(packet *Package) {
	user := packet.Local.String()
	switch packet.Type {
	case 0: //请求
		if remote, ok := Remotes.Load(user); ok {
			_, _ = remote.(*net.TCPConn).Write(packet.Data)
		} else {
			go RemoteChanIO(packet.Local, packet.Remote, packet)
		}
		break
	case 1: //响应
		if client, ok := Clients.Load(user); ok {
			_, _ = client.(*net.TCPConn).Write(packet.Data)
		}
		break
	case 0xC0: //心跳
		Timer = time.Now().Unix()
		break
	case 0xF0: //结束
		_ = (&os.Process{Pid: os.Getpid()}).Kill()
		break
	case 0xFF: //断开
		Clients.Delete(user)
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
		Clients.Store(client.RemoteAddr().String(), client)
		go ClientIO(client, remote)
	}
}

func ClientIO(client *net.TCPConn, remote *net.TCPAddr) {
	defer client.Close()
	user := client.RemoteAddr().String()
	local, _ := net.ResolveTCPAddr("tcp", user)
	buffer := make([]byte, BufferSize)
	for {
		if client, ok := Clients.Load(user); ok {
			count, err := client.(*net.TCPConn).Read(buffer)
			if err != nil {
				Way.SendPacket(&Package{
					Type:   0xff,
					Data:   []byte{},
					Local:  local,
					Remote: remote,
				})
				break
			}
			Way.SendPacket(&Package{
				Type:   0,
				Data:   buffer[:count],
				Local:  local,
				Remote: remote,
			})
		} else {
			break
		}
	}
}
