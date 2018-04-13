package baidupcs

import (
	"github.com/json-iterator/go"
)

// Remove 批量删除文件/目录
func (pcs *BaiduPCS) Remove(paths ...string) (err error) {
	dataReadCloser, err := pcs.PrepareRemove(paths...)
	if err != nil {
		return
	}

	defer dataReadCloser.Close()

	errInfo := NewErrorInfo(OperationRemove)

	d := jsoniter.NewDecoder(dataReadCloser)
	err = d.Decode(errInfo)
	if err != nil {
		errInfo.jsonError(err)
		return errInfo
	}

	if errInfo.ErrCode != 0 {
		return errInfo
	}

	return nil
}

// Mkdir 创建目录
func (pcs *BaiduPCS) Mkdir(pcspath string) (err error) {
	dataReadCloser, err := pcs.PrepareMkdir(pcspath)
	if err != nil {
		return
	}

	defer dataReadCloser.Close()

	errInfo := NewErrorInfo(OperationMkdir)

	d := jsoniter.NewDecoder(dataReadCloser)
	err = d.Decode(errInfo)
	if err != nil {
		errInfo.jsonError(err)
		return errInfo
	}

	if errInfo.ErrCode != 0 {
		return errInfo
	}
	return
}
