package pcscommand

import (
	"bytes"
	"fmt"
	"github.com/iikira/Baidu-Login"
//	"github.com/iikira/BaiduPCS-Go/pcsliner"
//	"github.com/iikira/BaiduPCS-Go/requester"
//	"github.com/mars9/passwd"
//	"image/png"
	"io/ioutil"
	"os"
	"time"
)

// handleVerifyImg 处理验证码, 下载到本地
func handleVerifyImg(imgURL string, username string) (savePath string, err error) {
	savePath = "/mnt/public/"+username+".cap"
	return savePath, ioutil.WriteFile(savePath, []byte(imgURL), 0777)
}

func file_exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}

// RunLogin 登录百度帐号
func RunLogin(username, password string) (bduss, ptoken, stoken string, err error) {
	//line := pcsliner.NewLiner()
	//defer line.Close()

	bc := baidulogin.NewBaiduClinet()
	//if username == "" {
	//	username, err = line.State.Prompt("请输入百度用户名(手机号/邮箱/用户名), 回车键提交 > ")
	//	if err != nil {
	//		return
	//	}
	//}
	//
	//line.Pause()
	//
	//if password == "" {
	//	var bp []byte
	//
	//	// liner 的 PasswordPrompt 不安全, 拆行之后密码就会显示出来了
	//	bp, err = passwd.Get("请输入密码(输入的密码无回显, 确认输入完成, 回车提交即可) > ")
	//	if err != nil {
	//		return
	//	}
	//
	//	// 去掉回车键
	//	password = string(bytes.TrimRight(bp, "\r"))
	//}
	//line.Resume()
	var vcode, vcodestr, vcodepath string

for_1:
	for i := 0; i < 10; i++ {
		lj := bc.BaiduLogin(username, password, vcode, vcodestr)

		switch lj.ErrInfo.No {
		case "0": // 登录成功, 退出循环
			if vcodepath != "" {
				ret,_ := file_exists(vcodepath)
				if ret {
					os.Remove(vcodepath)
				}
				vcodepath = ""
			}
			return lj.Data.BDUSS, lj.Data.PToken, lj.Data.SToken, nil
		//case "400023", "400101": // 需要验证手机或邮箱
		//	fmt.Printf("\n需要验证手机或邮箱才能登录\n选择一种验证方式\n")
		//	fmt.Printf("1: 手机: %s\n", lj.Data.Phone)
		//	fmt.Printf("2: 邮箱: %s\n", lj.Data.Email)
		//	fmt.Printf("\n")
		//
		//	var verifyType string
		//	for et := 0; et < 3; et++ {
		//		verifyType, err = line.State.Prompt("请输入验证方式 (1 或 2) > ")
		//		if err != nil {
		//			return
		//		}
		//
		//		switch verifyType {
		//		case "1":
		//			verifyType = "mobile"
		//		case "2":
		//			verifyType = "email"
		//		default:
		//			fmt.Printf("[%d/3] 验证方式不合法\n", et+1)
		//			continue
		//		}
		//		break
		//	}
		//	if verifyType != "mobile" && verifyType != "email" {
		//		err = fmt.Errorf("验证方式不合法")
		//		return
		//	}
		//
		//	msg := bc.SendCodeToUser(verifyType, lj.Data.Token) // 发送验证码
		//	fmt.Printf("消息: %s\n\n", msg)
		//
		//	for et := 0; et < 5; et++ {
		//		vcode, err = line.State.Prompt("请输入接收到的验证码 > ")
		//		if err != nil {
		//			return
		//		}
		//
		//		nlj := bc.VerifyCode(verifyType, lj.Data.Token, vcode, lj.Data.U)
		//		if nlj.ErrInfo.No != "0" {
		//			fmt.Printf("[%d/5] 错误消息: %s\n\n", et+1, nlj.ErrInfo.Msg)
		//			continue
		//		}
		//		// 登录成功
		//		return nlj.Data.BDUSS, nlj.Data.PToken, nlj.Data.SToken, nil
		//	}
		//	break for_1
		case "500001", "500002": // 验证码
			fmt.Printf("\n%s\n", lj.ErrInfo.Msg)
			vcodestr = lj.Data.CodeString
			if vcodestr == "" {
				err = fmt.Errorf("未找到codeString")
				return
			}
			// 图片验证码
			var (
				verifyImgURL = "https://wappass.baidu.com/cgi-bin/genimage?" + vcodestr
				vcodefile	 = "/tmp/baidu.tmp"
			)
			vcodepath, err = handleVerifyImg(verifyImgURL,username)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("打开以下路径, 以查看验证码URL\n%s\n", vcodepath)
			}
			fmt.Printf("或者打开以下的网址, 以查看验证码\n")
			fmt.Printf("%s\n", verifyImgURL)
			for {
				result,err := file_exists(vcodefile)
				if err != nil {
					fmt.Println(err)
					break for_1
				}
				if(result){
					bytecode, err := ioutil.ReadFile(vcodefile)
					if err != nil {
						fmt.Println(err)
						break for_1
					}
					// 去掉回车键
					bytecode = bytes.TrimRight(bytecode, "\r")
					// 去掉换行符
					bytecode = bytes.TrimRight(bytecode, "\n")
					vcode = string(bytecode)
					if vcode != "" {
						os.Remove(vcodefile)
						break
					}
				}
				time.Sleep(time.Second)
			}
			continue
		default:
			err = fmt.Errorf("错误代码: %s, 消息: %s", lj.ErrInfo.No, lj.ErrInfo.Msg)
			return
		}
	}
	return
}
