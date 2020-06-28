package main

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

// 8 version
// 1 type
// 4 sip
// 2 sport
// 4 eip
// 2 eport
// 4 length
// n data
type Package struct {
	Version [8]byte      // 协议版本 0xCC 0x05 0x20 0x13 0x14 0x00 0x00 0xCC
	Type    uint8        //包类型  0请求 1相应 2心跳 3断开
	Local   *net.TCPAddr //发起端地址
	Remote  *net.TCPAddr //接收端地址
	Data    []byte       // 数据
}

func IPAddrToInt(ip string) uint32 {
	scope := strings.Split(ip, ".")
	var result = 0
	var pos uint = 24
	for _, item := range scope {
		tmp, _ := strconv.Atoi(item)
		tmp = tmp << pos
		result = result | tmp
		pos -= 8
	}
	return uint32(result)
}
func IPAddrFromInt(ip uint32) string {
	scope := make([]string, 4)
	var length = len(scope)
	buffer := bytes.NewBufferString("")
	for i := 0; i < length; i++ {
		tempInt := ip & 0xFF
		scope[length-i-1] = strconv.Itoa(int(tempInt))
		ip = ip >> 8
	}
	for i := 0; i < length; i++ {
		buffer.WriteString(scope[i])
		if i < length-1 {
			buffer.WriteString(".")
		}
	}
	return buffer.String()
}

func (packet *Package) Pack(writer *bufio.Writer) error {
	packet.Version = Config.Version
	inner := func(data []byte) error {
		var err error
		err = binary.Write(writer, binary.BigEndian, packet.Version)
		err = binary.Write(writer, binary.BigEndian, packet.Type)
		err = binary.Write(writer, binary.BigEndian, uint32(IPAddrToInt(packet.Local.IP.String())))
		err = binary.Write(writer, binary.BigEndian, uint16(packet.Local.Port))
		err = binary.Write(writer, binary.BigEndian, uint32(IPAddrToInt(packet.Remote.IP.String())))
		err = binary.Write(writer, binary.BigEndian, uint16(packet.Remote.Port))
		err = binary.Write(writer, binary.BigEndian, int32(len(data)))
		err = binary.Write(writer, binary.BigEndian, data)
		return err
	}
	//长度大于 64*1024 需要切分
	total := len(packet.Data)
	maxsize := 64*1024 - 32
	if total > maxsize {
		scope := total / maxsize
		if total%maxsize > 0 {
			scope++
		}
		for i := 0; i < scope; i++ {
			if i*maxsize+maxsize > total {
				_ = inner(packet.Data[i*maxsize:])
			} else {
				_ = inner(packet.Data[i*maxsize : i*maxsize+maxsize])
			}
		}
		return writer.Flush()
	} else {
		_ = inner(packet.Data)
		return writer.Flush()
	}
}
func (packet *Package) UnPack(reader io.Reader) error {
	packet.Version = Config.Version
	var err error
	err = binary.Read(reader, binary.BigEndian, &packet.Version)
	err = binary.Read(reader, binary.BigEndian, &packet.Type)
	var localIP uint32 = 0
	var localPort uint16 = 0
	err = binary.Read(reader, binary.BigEndian, &localIP)
	err = binary.Read(reader, binary.BigEndian, &localPort)
	packet.Local, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", IPAddrFromInt(localIP), localPort))
	var remoteIP uint32 = 0
	var remotePort uint16 = 0
	err = binary.Read(reader, binary.BigEndian, &remoteIP)
	err = binary.Read(reader, binary.BigEndian, &remotePort)
	packet.Remote, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", IPAddrFromInt(remoteIP), remotePort))
	var length int32 = 0
	err = binary.Read(reader, binary.BigEndian, &length)
	packet.Data = make([]byte, length)
	err = binary.Read(reader, binary.BigEndian, &packet.Data)
	return err
}

type PackageQueue struct {
	Lock sync.Mutex
	Data *list.List
}

func (pq *PackageQueue) PushOne(v *Package) {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
	pq.Data.PushFront(v)
}
func (pq *PackageQueue) Push(v []*Package) {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
	for _, item := range v {
		pq.Data.PushFront(item)
	}
}
func (pq *PackageQueue) Pop() []*Package {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
	total := pq.Data.Len()
	var result []*Package
	for i := 0; i < total; i++ {
		tmp := pq.Data.Back()
		v := tmp.Value
		pq.Data.Remove(tmp)
		result = append(result, v.(*Package))
	}
	if result == nil {
		return make([]*Package, 0)
	}
	return result
}
