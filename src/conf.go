package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/axgle/mahonia"
	"runtime"
)

type Conf struct {
	Local   string
	Remote  string
	Version [8]byte
	Link    []Link
}
type Link struct {
	Local   string
	Remote  string
}

func ConfDefault() Conf {
	var cf Conf
	f, _ := os.OpenFile("./pawer.toml", os.O_RDONLY, 0)
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	if runtime.GOOS == "windows" {
		enc := mahonia.NewDecoder("gbk")
		_, data, _ := enc.Translate(b, true)
		fmt.Println(string(data))
		_, err := toml.Decode(string(data), &cf)
		if err != nil {
			panic(err)
		}
	} else {
		_, err := toml.Decode(string(b), &cf)
		if err != nil {
			panic(err)
		}
	}
	return cf
}
