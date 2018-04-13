package downloader

import (
	"os"
	"sync/atomic"
	"time"
)

// blockMonitor 延迟监控各线程状态,
// 重设长时间无响应, 和下载速度为 0 的线程
// 动态分配新线程
func (der *Downloader) blockMonitor() <-chan struct{} {
	c := make(chan struct{})
	go func() {
		for {
			// 下载暂停, 不开启监控
			if der.status.paused {
				time.Sleep(2 * time.Second)
				continue
			}

			// 下载完毕, 线程全部完成下载任务, 发送结束信号
			if der.status.BlockList.isAllDone() {
				if !der.Config.Testing {
					os.Remove(der.Config.SavePath + DownloadingFileSuffix) // 删除断点信息
				}

				c <- struct{}{}
				return
			}

			if !der.Config.Testing {
				der.recordBreakPoint()
			}

			// 获取下载速度
			speeds := der.status.StatusStat.speedsStat.GetSpeedsPerSecond()
			atomic.StoreInt64(&der.status.StatusStat.Speeds, speeds)
			if speeds > atomic.LoadInt64(&der.status.StatusStat.maxSpeeds) {
				atomic.StoreInt64(&der.status.StatusStat.maxSpeeds, speeds)
			}

			// 统计各线程的速度
			go func() {
				for k := range der.status.BlockList {
					go func(k int) {
						block := der.status.BlockList[k]
						time.Sleep(1 * time.Second)
						atomic.StoreInt64(&block.speed, block.speedsStat.GetSpeedsPerSecond())
					}(k)
				}
			}()

			// 速度减慢, 开启监控
			if atomic.LoadInt64(&der.status.StatusStat.Speeds) < atomic.LoadInt64(&der.status.StatusStat.maxSpeeds)/10 {
				atomic.StoreInt64(&der.status.StatusStat.maxSpeeds, 0)
				for k := range der.status.BlockList {
					go func(k int) {
						// 重设长时间无响应, 和下载速度为 0 的线程, 忽略正在写入数据到硬盘的
						// 过滤速度有变化的线程
						if der.status.BlockList[k].waitToWrite || atomic.LoadInt64(&der.status.BlockList[k].speed) != 0 {
							return
						}

						// 重设连接
						if r := der.status.BlockList[k].resp; r != nil {
							r.Body.Close()
							verbosef("MONITER: thread reload, thread id: %d\n", k)
						}

					}(k)

					// 动态分配新线程
					go func(k int) {
						der.monitorMu.Lock()

						// 筛选空闲的线程
						index, ok := der.status.BlockList.avaliableThread()
						if !ok { // 没有空的
							der.monitorMu.Unlock() // 解锁
							return
						}

						end := atomic.LoadInt64(&der.status.BlockList[k].End)
						middle := (atomic.LoadInt64(&der.status.BlockList[k].Begin) + end) / 2

						if end-middle <= MinParallelSize { // 如果线程剩余的下载量太少, 不分配空闲线程
							der.monitorMu.Unlock()
							return
						}

						// 折半
						atomic.StoreInt64(&der.status.BlockList[index].Begin, middle+1)
						atomic.StoreInt64(&der.status.BlockList[index].End, end)

						der.status.BlockList[index].IsFinal = der.status.BlockList[k].IsFinal
						atomic.StoreInt64(&der.status.BlockList[k].End, middle)

						// End 已变, 取消 Final
						der.status.BlockList[k].IsFinal = false

						der.monitorMu.Unlock()

						verbosef("MONITER: thread copied: %d -> %d\n", k, index)
						go der.addExecBlock(index)
					}(k)
				}
			}

			time.Sleep(1 * time.Second) // 监测频率 1 秒
		}
	}()
	return c
}
