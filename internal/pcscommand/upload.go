package pcscommand

import (
	"bytes"
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/iikira/BaiduPCS-Go/baidupcs"
	"github.com/iikira/BaiduPCS-Go/downloader/cachepool"
	"github.com/iikira/BaiduPCS-Go/pcscache"
	"github.com/iikira/BaiduPCS-Go/pcsutil"
	"github.com/iikira/BaiduPCS-Go/requester"
	"github.com/iikira/BaiduPCS-Go/requester/multipartreader"
	"github.com/iikira/BaiduPCS-Go/uploader"
	"hash"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const requiredSliceLen = 256 * pcsutil.KB // 256 KB

type utask struct {
	ListTask
	uploadInfo *LocalPathInfo // 要上传的本地文件详情
	savePath   string
}

// SumOption 计算文件摘要值配置
type SumOption struct {
	IsMD5Sum      bool
	IsSliceMD5Sum bool
	IsCRC32Sum    bool
}

// LocalPathInfo 本地文件详情
type LocalPathInfo struct {
	Path string // 本地路径

	Length   int64  // 文件大小
	SliceMD5 []byte // 文件前 requiredSliceLen (256KB) 切片的 md5 值
	MD5      []byte // 文件的 md5
	CRC32    uint32 // 文件的 crc32

	buf  []byte
	file *os.File // 文件
}

// OpenPath 检查文件状态并获取文件的大小 (Length)
func (lp *LocalPathInfo) OpenPath() bool {
	if lp.file != nil {
		lp.file.Close()
	}

	var err error
	lp.file, err = os.Open(lp.Path)
	if err != nil {
		return false
	}

	info, _ := lp.file.Stat()
	lp.Length = info.Size()
	return true
}

// Close 关闭文件
func (lp *LocalPathInfo) Close() error {
	if lp.file == nil {
		return fmt.Errorf("file is nil")
	}

	return lp.file.Close()
}

func (lp *LocalPathInfo) repeatRead(ws ...io.Writer) {
	if lp.file == nil {
		return
	}

	if lp.buf == nil {
		lp.buf = cachepool.SetIfNotExist(0, int(requiredSliceLen))
	}

	var (
		begin int64
		n     int
		err   error
	)

	handle := func() {
		begin += int64(n)
		for k := range ws {
			ws[k].Write(lp.buf[:n])
		}
	}

	// 读文件
	for {
		n, err = lp.file.ReadAt(lp.buf, begin)
		if err != nil {
			if err == io.EOF {
				handle()
			} else {
				fmt.Printf("%s\n", err)
			}
			break
		}

		handle()
	}
}

// Sum 计算文件摘要值
func (lp *LocalPathInfo) Sum(opt SumOption) {
	var (
		md5w   hash.Hash
		crc32w hash.Hash32
	)

	ws := make([]io.Writer, 0, 2)
	if opt.IsMD5Sum {
		md5w = md5.New()
		ws = append(ws, md5w)
	}
	if opt.IsCRC32Sum {
		crc32w = crc32.NewIEEE()
		ws = append(ws, crc32w)
	}
	if opt.IsSliceMD5Sum {
		lp.SliceMD5Sum()
	}

	lp.repeatRead(ws...)

	if opt.IsMD5Sum {
		lp.MD5 = md5w.Sum(nil)
	}
	if opt.IsCRC32Sum {
		lp.CRC32 = crc32w.Sum32()
	}
}

// Md5Sum 获取文件的 md5 值
func (lp *LocalPathInfo) Md5Sum() {
	lp.Sum(SumOption{
		IsMD5Sum: true,
	})
}

// SliceMD5Sum 获取文件前 requiredSliceLen (256KB) 切片的 md5 值
func (lp *LocalPathInfo) SliceMD5Sum() {
	if lp.file == nil {
		return
	}

	// 获取前 256KB 文件切片的 md5
	if lp.buf == nil {
		lp.buf = cachepool.SetIfNotExist(0, int(requiredSliceLen))
	}

	m := md5.New()
	n, err := lp.file.ReadAt(lp.buf, 0)
	if err != nil {
		if err == io.EOF {
			goto md5sum
		} else {
			fmt.Printf("SliceMD5Sum: %s\n", err)
			return
		}
	}

md5sum:
	m.Write(lp.buf[:n])
	lp.SliceMD5 = m.Sum(nil)
}

// Crc32Sum 获取文件的 crc32 值
func (lp *LocalPathInfo) Crc32Sum() {
	lp.Sum(SumOption{
		IsCRC32Sum: true,
	})
}

// RunRapidUpload 执行秒传文件, 前提是知道文件的大小, md5, 前256KB切片的 md5, crc32
func RunRapidUpload(targetPath, contentMD5, sliceMD5, crc32 string, length int64) {
	targetPath, err := getAbsPath(targetPath)
	if err != nil {
		fmt.Printf("警告: 尝试秒传文件, 获取网盘路径 %s 错误, %s\n", targetPath, err)
	}

	if sliceMD5 == "" {
		sliceMD5 = "ec87a838931d4d5d2e94a04644788a55" // 长度为32
	}

	err = info.RapidUpload(targetPath, contentMD5, sliceMD5, crc32, length)
	if err != nil {
		fmt.Printf("秒传失败, 消息: %s\n", err)
		return
	}

	fmt.Printf("秒传成功, 保存到网盘路径: %s\n", targetPath)
	return
}

// RunUpload 执行文件上传
func RunUpload(localPaths []string, savePath string) {
	absSavePath, err := getAbsPath(savePath)
	if err != nil {
		fmt.Printf("警告: 上传文件, 获取网盘路径 %s 错误, %s\n", savePath, err)
	}

	switch len(localPaths) {
	case 0:
		fmt.Printf("本地路径为空\n")
		return
	}

	var (
		ulist         = list.New()
		lastID        int
		globedPathDir string
		subSavePath   string
	)

	for k := range localPaths {
		globedPaths, err := filepath.Glob(localPaths[k])
		if err != nil {
			fmt.Printf("上传文件, 匹配本地路径失败, %s\n", err)
			continue
		}

		for k2 := range globedPaths {
			walkedFiles, err := pcsutil.WalkDir(globedPaths[k2], "")
			if err != nil {
				fmt.Printf("警告: %s\n", err)
				continue
			}

			for k3 := range walkedFiles {
				// 针对 windows 的目录处理
				if os.PathSeparator == '\\' {
					walkedFiles[k3] = pcsutil.ConvertToUnixPathSeparator(walkedFiles[k3])
					globedPathDir = pcsutil.ConvertToUnixPathSeparator(filepath.Dir(globedPaths[k2]))
				} else {
					globedPathDir = filepath.Dir(globedPaths[k2])
				}

				subSavePath = strings.TrimPrefix(walkedFiles[k3], globedPathDir)

				lastID++
				ulist.PushBack(&utask{
					ListTask: ListTask{
						ID:       lastID,
						MaxRetry: 3,
					},
					uploadInfo: &LocalPathInfo{
						Path: walkedFiles[k3],
					},
					savePath: path.Clean(absSavePath + "/" + subSavePath),
				})

				fmt.Printf("[%d] 加入上传队列: %s\n", lastID, walkedFiles[k3])
			}
		}
	}

	if lastID == 0 {
		fmt.Printf("未检测到上传的文件, 请检查文件路径或通配符是否正确.\n")
		return
	}

	var (
		e             *list.Element
		task          *utask
		msg			  string
		handleTaskErr = func(task *utask, errManifest string, err error) {
			if task == nil {
				panic("task is nil")
			}

			if err == nil {
				return
			}

			// 不重试的情况
			switch {
			case strings.Contains(err.Error(), baidupcs.StrRemoteError):
				msg = fmt.Sprintf("[%d] %s, %s\n", task.ID, errManifest, err)
				fmt.Print(msg)
				pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
				return
			}
			msg = fmt.Sprintf("[%d] %s, %s, 重试 %d/%d\n", task.ID, errManifest, err, task.retry, task.MaxRetry)
			fmt.Print(msg)
			pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)

			// 未达到失败重试最大次数, 将任务推送到队列末尾
			if task.retry < task.MaxRetry {
				task.retry++
				ulist.PushBack(task)
				time.Sleep(3 * time.Duration(task.retry) * time.Second)
			} else {
				task.uploadInfo.Close() // 关闭文件
			}
		}
		totalSize int64
	)

	for {
		e = ulist.Front()
		if e == nil { // 结束
			break
		}

		ulist.Remove(e) // 载入任务后, 移除队列

		task = e.Value.(*utask)
		if task == nil {
			continue
		}

		msg = fmt.Sprintf("[%d] 准备上传: %s\n", task.ID, task.uploadInfo.Path)
		fmt.Print(msg)
		pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)

		if !task.uploadInfo.OpenPath() {
			msg = fmt.Sprintf("[%d] 文件不可读, 跳过...\n", task.ID)
			fmt.Print(msg)
			pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
			task.uploadInfo.Close()
			continue
		}

		panDir, panFile := path.Split(task.savePath)

		// 设置缓存
		if !pcscache.DirCache.Existed(panDir) {
			fdl, err := info.FilesDirectoriesList(panDir, false)
			if err == nil {
				pcscache.DirCache.Set(panDir, &fdl)
			}
		}

		if task.uploadInfo.Length >= 128*pcsutil.MB {
			msg = fmt.Sprintf("[%d] 检测秒传中, 请稍候...\n", task.ID)
			fmt.Print(msg)
			pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
		}

		task.uploadInfo.Md5Sum()

		// 检测缓存, 通过文件的md5值判断本地文件和网盘文件是否一样
		fd := pcscache.DirCache.FindFileDirectory(panDir, panFile)
		if fd != nil {
			decodedMD5, _ := hex.DecodeString(fd.MD5)
			if bytes.Compare(decodedMD5, task.uploadInfo.MD5) == 0 {
				msg = fmt.Sprintf("[%d] 目标文件, %s, 已存在, 跳过...\n", task.ID, task.savePath)
				fmt.Print(msg)
				pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
				continue
			}
		}

		// 文件大于256kb, 应该要检测秒传, 反之则不应检测秒传
		// 经测试, 秒传文件并非一定要大于256KB
		if task.uploadInfo.Length >= requiredSliceLen {
			// do nothing
		}

		// 经过测试, 秒传文件并非需要前256kb切片的md5值, 只需格式符合即可
		task.uploadInfo.SliceMD5Sum()

		// 经测试, 文件的 crc32 值并非秒传文件所必需
		// task.uploadInfo.crc32Sum()

		err := info.RapidUpload(task.savePath, hex.EncodeToString(task.uploadInfo.MD5), hex.EncodeToString(task.uploadInfo.SliceMD5), fmt.Sprint(task.uploadInfo.CRC32), task.uploadInfo.Length)
		if err == nil {
			msg = fmt.Sprintf("[%d] 秒传成功, 保存到网盘路径: %s\n", task.ID, task.savePath)
			fmt.Print(msg)
			pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)

			task.uploadInfo.Close() // 关闭文件
			totalSize += task.uploadInfo.Length
			continue
		}

		msg = fmt.Sprintf("[%d] 秒传失败, 开始上传文件...\n", task.ID)
		fmt.Print(msg)
		pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)

		// 秒传失败, 开始上传文件
		err = info.Upload(task.savePath, func(uploadURL string, jar *cookiejar.Jar) (resp *http.Response, uperr error) {
			h := requester.NewHTTPClient()
			h.SetCookiejar(jar)

			u := uploader.NewUploader(uploadURL, multipartreader.NewFileReadedLen64(task.uploadInfo.file), &uploader.Options{
				IsMultiPart: true,
				Client:      h,
			})

			exit := make(chan struct{})

			u.OnExecute(func() {
				ulog := fmt.Sprintf("%s/%d.log", pcsutil.CheckLogPath(), task.ID)
				msg = fmt.Sprintf("[%d] %s\n", task.ID, ulog)
				fmt.Print(msg)
				pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
				for {
					select {
					case v, ok := <-u.UploadStatus:
						if !ok {
							return
						}

						if v.Length == 0 {
							fmt.Printf("\r[%d] Prepareing upload...", task.ID)
							continue
						}

						msg = fmt.Sprintf("\r[%d] ↑ %s/%s %s/s in %s ............", task.ID,
							pcsutil.ConvertFileSize(v.Uploaded, 2),
							pcsutil.ConvertFileSize(v.Length, 2),
							pcsutil.ConvertFileSize(v.Speed, 2),
							v.TimeElapsed,
						)
						fmt.Print(msg)
						pcsutil.WriteLog(ulog, msg, true)
					}
				}
			})

			u.OnFinish(func() {
				exit <- struct{}{}
			})

			<-u.Execute(func(upresp *http.Response, err error) {
				resp = upresp
				uperr = err
			})

			<-exit
			close(exit)
			return
		})

		fmt.Printf("\n")

		if err != nil {
			handleTaskErr(task, "上传文件失败", err)
			continue
		}

		msg = fmt.Sprintf("[%d] 上传文件成功, 保存到网盘路径: %s\n", task.ID, task.savePath)
		fmt.Print(msg)
		pcsutil.WriteLog(pcsutil.CheckBaiduLog(), msg, false)
		task.uploadInfo.Close() // 关闭文件
		totalSize += task.uploadInfo.Length
	}

	fmt.Printf("\n")
	fmt.Printf("全部上传完毕, 总大小: %s\n", pcsutil.ConvertFileSize(totalSize))
}

// GetFileSum 获取文件的大小, md5, 前256KB切片的 md5, crc32
func GetFileSum(localPath string, opt *SumOption) (lp *LocalPathInfo, err error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if fileStat.IsDir() {
		return nil, fmt.Errorf("sum %s: is a directory", localPath)
	}

	lp = &LocalPathInfo{
		Path:   localPath,
		file:   file,
		Length: fileStat.Size(),
	}

	lp.Sum(*opt)

	return lp, nil
}
