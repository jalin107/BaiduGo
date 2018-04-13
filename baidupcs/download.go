package baidupcs

import (
	"github.com/iikira/BaiduPCS-Go/internal/pcsconfig"
	"net/http/cookiejar"
)

// DownloadFunc 下载文件处理函数
type DownloadFunc func(downloadURL string, jar *cookiejar.Jar, savePath string) error

// DownloadFile 下载单个文件
func (pcs *BaiduPCS) DownloadFile(path string, downloadFunc DownloadFunc, savePath string) (err error) {
	pcs.setPCSURL("file", "download", map[string]string{
		"path": path,
	})
	return downloadFunc(pcs.url.String(), pcs.client.Jar.(*cookiejar.Jar), pcsconfig.GetSavePath(savePath,path))
}

// DownloadStreamFile 下载流式文件
func (pcs *BaiduPCS) DownloadStreamFile(path string, downloadFunc DownloadFunc, savePath string) (err error) {
	pcs.setPCSURL("stream", "download", map[string]string{
		"path": path,
	})

	return downloadFunc(pcs.url.String(), pcs.client.Jar.(*cookiejar.Jar), pcsconfig.GetSavePath(savePath,path))
}
