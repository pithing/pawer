package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"time"
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
	//解决链接问题
	go func() {
		var err error
		for {
			if way.RemoteAddr.String() == "0.0.0.0:0" {
				//不链接远端，接收到conn后使用此链接发送和接收数据
				way.remote = way.local
				if way.remote == nil {
					time.Sleep(time.Second)
					continue
				}
			}
			if reflect.ValueOf(way.remote).IsNil() || err != nil {
				way.remote, err = net.DialTCP("tcp", nil, way.RemoteAddr)
			}
			if err == nil {
				err = way.remote.SetKeepAlive(true)
			}
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		var local net.Conn
		var listener *net.TCPListener
		var err error
		if way.LocalAddr.String() != "0.0.0.0:0" {
			listener, err = net.ListenTCP("tcp", way.LocalAddr)
			if err != nil {
				panic(err)
			}
		}
		for {
			if way.LocalAddr.String() != "0.0.0.0:0" {
				way.local, err = listener.AcceptTCP()
				local = way.local
			} else {
				//本地不监听，主动链接后使用链接的conn发送和接收数据
				local = way.remote
			}
			if err != nil || reflect.ValueOf(local).IsNil() {
				time.Sleep(time.Second)
				continue
			}
			scanner := bufio.NewScanner(local)
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
				err = packet.UnPack(bytes.NewReader(scanner.Bytes()))
				if way.ReceiveAction != nil {
					way.ReceiveAction(packet)
				}
			}
			if err := scanner.Err(); err != nil {
				_ = local.Close()
			}
		}
	}()
}

//发送数据包
func (way *OneWay) SendPacket(packet *Package) {
	packet.Pack(bufio.NewWriter(way.remote))
}
