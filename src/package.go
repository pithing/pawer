package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

var Config = ConfDefault()

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
	Type    int8         //包类型  0请求 1相应 2心跳 3断开
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
	var len = len(scope)
	buffer := bytes.NewBufferString("")
	for i := 0; i < len; i++ {
		tempInt := ip & 0xFF
		scope[len-i-1] = strconv.Itoa(int(tempInt))
		ip = ip >> 8
	}
	for i := 0; i < len; i++ {
		buffer.WriteString(scope[i])
		if i < len-1 {
			buffer.WriteString(".")
		}
	}
	return buffer.String()
}

func (packet *Package) Pack(writer *bufio.Writer) error {
	var err error
	err = binary.Write(writer, binary.BigEndian, packet.Version)
	err = binary.Write(writer, binary.BigEndian, packet.Type)
	err = binary.Write(writer, binary.BigEndian, uint32(IPAddrToInt(packet.Local.IP.String())))
	err = binary.Write(writer, binary.BigEndian, uint16(packet.Local.Port))
	err = binary.Write(writer, binary.BigEndian, uint32(IPAddrToInt(packet.Remote.IP.String())))
	err = binary.Write(writer, binary.BigEndian, uint16(packet.Remote.Port))
	err = binary.Write(writer, binary.BigEndian, int32(len(packet.Data)))
	err = binary.Write(writer, binary.BigEndian, packet.Data)
	err = binary.Write(writer, binary.BigEndian, packet.Data)
	err = writer.Flush()
	return err
}
func (packet *Package) UnPack(reader io.Reader) error {
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
