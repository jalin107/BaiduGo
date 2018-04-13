package baidupcs

import (
	"fmt"
	"github.com/iikira/BaiduPCS-Go/internal/pcsconfig"
	"github.com/iikira/BaiduPCS-Go/requester"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

const (
	// OperationQuotaInfo 获取当前用户空间配额信息
	OperationQuotaInfo = "获取当前用户空间配额信息"
	// OperationFilesDirectoriesMeta 获取文件/目录的元信息
	OperationFilesDirectoriesMeta = "获取文件/目录的元信息"
	// OperationFilesDirectoriesList 获取目录下的文件列表
	OperationFilesDirectoriesList = "获取目录下的文件列表"
	// OperationRemove 删除文件/目录
	OperationRemove = "删除文件/目录"
	// OperationMkdir 创建目录
	OperationMkdir = "创建目录"
	// OperationRename 重命名文件/目录
	OperationRename = "重命名文件/目录"
	// OperationCopy 拷贝文件/目录
	OperationCopy = "拷贝文件/目录"
	// OperationMove 移动文件/目录
	OperationMove = "移动文件/目录"
	// OperationRapidUpload 秒传文件
	OperationRapidUpload = "秒传文件"
	// OperationUpload 上传单个文件
	OperationUpload = "上传单个文件"
	// OperationUploadTmpFile 分片上传—文件分片及上传
	OperationUploadTmpFile = "分片上传—文件分片及上传"
	// OperationUploadCreateSuperFile 分片上传—合并分片文件
	OperationUploadCreateSuperFile = "分片上传—合并分片文件"
	// OperationFileDownload 下载单个文件
	OperationFileDownload = "下载单个文件"
	// OperationStreamFileDownload 下载流式文件
	OperationStreamFileDownload = "下载流式文件"
	// OperationCloudDlAddTask 添加离线下载任务
	OperationCloudDlAddTask = "添加离线下载任务"
	// OperationCloudDlQueryTask 精确查询离线下载任务
	OperationCloudDlQueryTask = "精确查询离线下载任务"
	// OperationCloudDlListTask 查询离线下载任务列表
	OperationCloudDlListTask = "查询离线下载任务列表"
	// OperationCloudDlCancelTask 取消离线下载任务
	OperationCloudDlCancelTask = "取消离线下载任务"
	// OperationCloudDlDeleteTask 删除离线下载任务
	OperationCloudDlDeleteTask = "删除离线下载任务"
)

var (
	// AppID 百度 PCS 应用 ID
	AppID int
)

// BaiduPCS 百度 PCS API 详情
type BaiduPCS struct {
	url    *url.URL
	client *requester.HTTPClient // http 客户端
}

// NewPCS 提供 百度BDUSS, 返回 PCSApi 指针对象
func NewPCS(bduss string) *BaiduPCS {
	client := requester.NewHTTPClient()
	client.UserAgent = pcsconfig.Config.UserAgent

	pcsURL := &url.URL{
		Scheme: "http",
		Host:   "pcs.baidu.com",
	}

	cookie := &http.Cookie{
		Name:  "BDUSS",
		Value: bduss,
	}

	jar, _ := cookiejar.New(nil)
	jar.SetCookies(pcsURL, []*http.Cookie{
		cookie,
	})
	jar.SetCookies((&url.URL{
		Scheme: "http",
		Host:   "pan.baidu.com",
	}), []*http.Cookie{
		cookie,
	})
	client.SetCookiejar(jar)

	return &BaiduPCS{
		url:    pcsURL,
		client: client,
	}
}

func (pcs *BaiduPCS) setPCSURL(subPath, method string, param ...map[string]string) {
	pcs.url = &url.URL{
		Scheme: "http",
		Host:   "pcs.baidu.com",
		Path:   "/rest/2.0/pcs/" + subPath,
	}

	uv := pcs.url.Query()
	uv.Set("app_id", fmt.Sprint(pcsconfig.Config.AppID))
	uv.Set("method", method)
	for k := range param {
		for k2 := range param[k] {
			uv.Set(k2, param[k][k2])
		}
	}

	pcs.url.RawQuery = uv.Encode()
}

func (pcs *BaiduPCS) setPCSURL2(subPath, method string, param ...map[string]string) {
	pcs.url = &url.URL{
		Scheme: "http",
		Host:   "pan.baidu.com",
		Path:   "/rest/2.0/" + subPath,
	}

	uv := pcs.url.Query()
	uv.Set("app_id", "250528")
	uv.Set("method", method)
	for k := range param {
		for k2 := range param[k] {
			uv.Set(k2, param[k][k2])
		}
	}

	pcs.url.RawQuery = uv.Encode()
}

