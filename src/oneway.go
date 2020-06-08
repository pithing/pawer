package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

type OneWay struct {
	//private
	local  *net.TCPConn //单向本地监听
	remote *net.TCPConn //单向远端链接
	//public
	LocalAddr     *net.TCPAddr //单向本地监听地址
	RemoteAddr    *net.TCPAddr //单向远端链接地址
	Version       [8]byte
	ReceiveAction ReceiveFunc //单向接收回调
}
type ReceiveFunc func(*Package)

func (way *OneWay) Default(localAddr string, remoteAddr string) {
	var err error
	way.LocalAddr, err = net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		panic(err)
	}
	way.RemoteAddr, err = net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		panic(err)
	}
}
func (way *OneWay) WayConnIO() {
	ScannerSync := func(conn *net.TCPConn) {
		scanner := bufio.NewScanner(conn)
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			header := len(way.Version) + 17
			if !atEOF && len(data) > header {
				index := bytes.Index(data, way.Version[:])
				if index < 0 {
					return
				}
				length := int32(0)
				_ = binary.Read(bytes.NewReader(data[index+header-4:index+header]), binary.BigEndian, &length)
				total := header + int(length)
				if index+total <= len(data) {
					return index + total, data[index : index+total], nil
				}
			}
			return
		})
		for scanner.Scan() {
			packet := new(Package)
			err := packet.UnPack(bytes.NewReader(scanner.Bytes()))
			if err != nil {
				break
			}
			if way.ReceiveAction != nil {
				way.ReceiveAction(packet)
			}
		}
		log.Printf(scanner.Err().Error())
		_ = conn.Close()
	}

	var err error
	var listener *net.TCPListener
	//解决链接问题
	if way.LocalAddr.String() == "0.0.0.0:0" {
		way.remote, err = net.DialTCP("tcp", nil, way.RemoteAddr)
		way.local = way.remote
	} else if way.RemoteAddr.String() == "0.0.0.0:0" {
		listener, err = net.ListenTCP("tcp", way.LocalAddr)
		way.local, err = listener.AcceptTCP()
		way.remote = way.local
	} else {
		listener, err = net.ListenTCP("tcp", way.LocalAddr)
		way.remote, err = net.DialTCP("tcp", nil, way.RemoteAddr)
		way.local, err = listener.AcceptTCP()
	}
	if err != nil {
		panic(err)
	}
	ScannerSync(way.local)
}

//发送数据包
func (way *OneWay) SendPacket(packet *Package) {
	packet.Pack(bufio.NewWriter(way.remote))
}
