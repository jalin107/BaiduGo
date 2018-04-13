package downloader

import (
	"fmt"
	"github.com/iikira/BaiduPCS-Go/pcsverbose"
	"os"
)

// checkFileExist 检查文件是否存在,
// 只有当文件存在, 断点续传文件不存在时, 才判断为存在
func checkFileExist(path string) (err error) {
	if _, err = os.Stat(path); err == nil {
		if _, err = os.Stat(path + DownloadingFileSuffix); err != nil {
			return fmt.Errorf("文件已存在: %s", path)
		}
	}

	return nil
}

func trigger(f func()) {
	if f == nil {
		return
	}

	go f()
}

func triggerOnError(f func(code int, err error), code int, err error) {
	if f == nil {
		return
	}

	go f(code, err)
}

func verbosef(format string, a ...interface{}) {
	if pcsverbose.IsVerbose {
		fmt.Printf(format, a...)
	}
}
