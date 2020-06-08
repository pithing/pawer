package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/axgle/mahonia"
)

type Conf struct {
	Local   string
	Remote  string
	Version [8]byte
	Link    []Link
}
type Link struct {
	Local  string
	Remote string
}

func ConfDefault() Conf {
	var cf Conf
	f, err := os.OpenFile(GetCurPath()+"/pawer.toml", os.O_RDONLY, 0)
	if err != nil {
		if runtime.GOOS == "windows" {
			f, err = os.OpenFile("C:/pawer.toml", os.O_RDONLY, 0)
		} else {
			f, err = os.OpenFile("/etc/pawer.toml", os.O_RDONLY, 0)
		}
		if err != nil {
			panic(err)
		}
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	if runtime.GOOS == "windows" {
		enc := mahonia.NewDecoder("gbk")
		_, data, _ := enc.Translate(b, true)
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
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	rst := filepath.Dir(path)
	return rst
}
