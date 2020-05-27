package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"time"
)

type OneWay struct {
	//private
	local  *net.TCPConn //单向本地监听
	remote *net.TCPConn //单向远端链接
	//public
	LocalAddr  *net.TCPAddr //单向本地监听地址
	RemoteAddr *net.TCPAddr //单向远端链接地址
	Version    [8]byte
	Sender     chan *Package //发送通道
	Reader     chan *Package //接收通道
}

func (way OneWay) Default(localAddr string, remoteAddr string) *OneWay {
	var err error
	way.LocalAddr, err = net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		panic(err)
	}
	way.RemoteAddr, err = net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		panic(err)
	}

	way.Sender = make(chan *Package)
	way.Reader = make(chan *Package)
	go way.readWayIo()
	go way.sendWayIo()
	return &way
}

//发送队列
func (way OneWay) sendWayIo() {
	var err error
	var writer *bufio.Writer
	for {
		select {
		case packet := <-way.Sender:
			for {
				var remote *net.TCPConn
				if way.RemoteAddr.String() == "0.0.0.0:0" {
					//不链接远端，接收到conn后使用此链接发送和接收数据
					remote = way.local
					if remote == nil {
						time.Sleep(time.Second)
						continue
					}
				} else {
					remote = way.remote
				}
				if remote == nil || err != nil {
					if remote != nil {
						_ = remote.Close()
					}
					way.remote, err = net.DialTCP("tcp", nil, way.RemoteAddr)
					if err != nil {
						continue
					}
					remote = way.remote
				}
				writer = bufio.NewWriter(way.remote)
				err = packet.Pack(writer)
				if err == nil {
					break
				}
			}
		}
	}
}

//监听队列
func (way OneWay) readWayIo() {
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
		if err != nil || local == nil {
			time.Sleep(time.Second)
			continue
		}
		scanner := bufio.NewScanner(local)
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			header := len(way.Version) + 13
			if !atEOF && len(data) > header {
				index := bytes.Index(data, way.Version[:])
				if index < 0 {
					return
				}
				length := int32(0)
				_ = binary.Read(bytes.NewReader(data[index+header:index+header+4]), binary.BigEndian, &length)
				total := header + 4 + int(length)
				if index+total <= len(data) {
					return index + total, data[index : index+total], nil
				}
			}
			return
		})
		for scanner.Scan() {
			packet := new(Package)
			err = packet.UnPack(bytes.NewReader(scanner.Bytes()))
			way.Reader <- packet
		}
		if err := scanner.Err(); err != nil {
			_ = local.Close()
		}
	}
}
