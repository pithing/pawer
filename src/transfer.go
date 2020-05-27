package main

import (
	"net"
	"os"
	"sync"
	"time"
)

type MapMutex struct {
	Data map[string]net.Conn
	Lock sync.RWMutex
}

func (d *MapMutex) Get(k string) (net.Conn, bool) {
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	item, has := d.Data[k]
	return item, has
}

func (d *MapMutex) Set(k string, v net.Conn) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Data[k] = v
}

func (d *MapMutex) Remove(k string) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	delete(d.Data, k)
}

var Timer int64 = time.Now().Unix()

var Clients MapMutex = MapMutex{Data: make(map[string]net.Conn), Lock: sync.RWMutex{}}
var Remotes MapMutex = MapMutex{Data: make(map[string]net.Conn), Lock: sync.RWMutex{}}

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
		remote, has := Remotes.Get(user)
		var err error
		var conn *net.TCPConn
		if !has {
			conn, err = net.DialTCP("tcp", nil, packet.Remote)
			remote = conn
			Remotes.Set(user, remote)
			//开启接收
			go func() {
				for {
					buffer := make([]byte, 1024)
					count, err := conn.Read(buffer)
					if err != nil {
						break
					}
					//发送报文信息
					Way.Sender <- &Package{
						Type:   1,
						Data:   buffer[:count],
						Local:  packet.Local,
						Remote: packet.Remote,
					}
				}
			}()
		}
		_, err = remote.Write(packet.Data)
		if err != nil {
			if remote != nil {
				_ = remote.Close()
			}
			Remotes.Remove(user)
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
		client, has := Clients.Get(user)
		if has {
			_, err := client.Write(packet.Data)
			if err != nil {
				_ = client.Close()
				Clients.Remove(user)
			}
		}
		break
	case 0xC0: //心跳
		Timer = time.Now().Unix()
		break
	case 0xF0: //结束
		_ = (&os.Process{Pid: os.Getpid()}).Kill()
		break
	case 0xFF: //断开
		client, has := Clients.Get(user)
		if has {
			_ = client.Close()
			Clients.Remove(user)
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
	user := client.RemoteAddr().String()
	localAddr, _ := net.ResolveTCPAddr("tcp", user)
	Clients.Set(user, client)
	buffer := make([]byte, 1024)
	for {
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
	Clients.Remove(user)
	//发送断开链接
	Way.Sender <- &Package{
		Type:   0xff,
		Data:   []byte{},
		Local:  localAddr,
		Remote: remoteAddr,
	}
}
