package main

import (
	"net"
	"os"
	"time"
)

var Timer int64
var Clients map[string]net.Conn = make(map[string]net.Conn)
var Remotes map[string]net.Conn = make(map[string]net.Conn)

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
		var err error
		if !has {
			conn, err := net.DialTCP("tcp", nil, packet.Remote)
			if err != nil {
				remote = conn
				Remotes[user] = remote
			}
		}
		_, err = remote.Write(packet.Data)
		if err != nil {
			if remote != nil {
				_ = remote.Close()
			}
			delete(Remotes, user)
			//发送断开链接
			Way.Sender <- &Package{
				Type:   0xff,
				Data:   []byte{},
				Local:  packet.Local,
				Remote: packet.Remote,
			}
		}
		break
	case 1: //响应
		client, has := Clients[user]
		if has {
			_, _ = client.Write(packet.Data)
		}
		break
	case 0xC0: //心跳
		Timer = time.Now().Unix()
		break
	case 0xF0: //结束
		_ = (&os.Process{Pid: os.Getpid()}).Kill()
		break
	case 0xFF: //断开
		client, has := Clients[user]
		if has {
			_ = client.Close()
			delete(Clients, user)
		}
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
		go ClientIO(client, remote)
	}
}

func ClientIO(client *net.TCPConn, remoteAddr *net.TCPAddr) {
	ipAddr := client.RemoteAddr().String()
	localAddr, _ := net.ResolveTCPAddr("tcp", ipAddr)
	Clients[ipAddr] = client
	for {
		buffer := make([]byte, 1024)
		count, err := client.Read(buffer)
		if err != nil {
			break
		}
		//发送报文信息
		Way.Sender <- &Package{
			Type:   0,
			Data:   buffer[:count],
			Local:  localAddr,
			Remote: remoteAddr,
		}
	}
	delete(Clients, ipAddr)
	//发送断开链接
	Way.Sender <- &Package{
		Type:   0xff,
		Data:   []byte{},
		Local:  localAddr,
		Remote: remoteAddr,
	}
}
