// Package pcsutil 工具包
package pcsutil

import (
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"fmt"
)

var (
	// PipeInput 命令中是否为管道输入
	PipeInput bool
)

func init() {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	PipeInput = (fileInfo.Mode() & os.ModeNamedPipe) == os.ModeNamedPipe
}

// GetURLCookieString 返回cookie字串
func GetURLCookieString(urlString string, jar *cookiejar.Jar) string {
	url, _ := url.Parse(urlString)
	cookies := jar.Cookies(url)
	cookieString := ""
	for _, v := range cookies {
		cookieString += v.String() + "; "
	}
	cookieString = strings.TrimRight(cookieString, "; ")
	return cookieString
}

// DecompressGZIP 对 io.Reader 数据, 进行 gzip 解压
func DecompressGZIP(r io.Reader) ([]byte, error) {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	gzipReader.Close()
	return ioutil.ReadAll(gzipReader)
}

// FlagProvided 检测命令行是否提供名为 name 的 flag, 支持多个name(names)
func FlagProvided(names ...string) bool {
	if len(names) == 0 {
		return false
	}
	var targetFlag *flag.Flag
	for _, name := range names {
		targetFlag = flag.Lookup(name)
		if targetFlag == nil {
			return false
		}
		if targetFlag.DefValue == targetFlag.Value.String() {
			return false
		}
	}
	return true
}

func CheckLogPath() string {
	var mainlog = "/tmp/baidu"
	if !IsFileExist(mainlog) {
		os.Mkdir(mainlog, 0777)
	}
	path := fmt.Sprintf("%s/%d", mainlog, os.Getpid())
	if !IsFileExist(path) {
		os.Mkdir(path, 0777)
	}
	return path
}

func CheckBaiduLog() string {
	var mainlog = "/tmp/baidu"
	if !IsFileExist(mainlog) {
		os.Mkdir(mainlog, 0777)
	}
	path := fmt.Sprintf("%s/baidu-%d.log", mainlog, os.Getpid())
	return path
}

func WriteLog(filename string, msg string, wct bool) (error) {
	var (
		f *os.File
		err error
	)
	if IsFileExist(filename) {
		if wct==true { //打开并清空文件
			f, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		}else{//追加写入
			f, err = os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0666)
		}
	}else{
		f, err = os.Create(filename)
	}
	if err != nil { return err }
	n,_ := f.Seek(-1024, os.SEEK_END)
	f.WriteAt([]byte(msg), n)
	f.Close()
	return nil
}

func IsFileExist(path string) (bool) {
	_, err := os.Stat(path)
	if err == nil { return true }
	if os.IsNotExist(err) { return false }
	return true
}