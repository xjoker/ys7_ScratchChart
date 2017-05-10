/*
	调用萤石云接口，获得特定序列号摄像机的截图
	存储为一张最新图，并在程序同路径下的img内存储历史所有图片

	调用参数
	-appKey		openapi的appkey
	-appSecret	openapi的appSecret
	-nowpath	最新图片存放位置
	-sn			抓取设备的序列号
	-interval	抓取图片的间隔
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

func getToken(key, sec string) (string, int64) {
	req, err := http.NewRequest("POST", "https://open.ys7.com/api/lapp/token/get", strings.NewReader("appKey="+key+"&appSecret="+sec))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
	json, err := simplejson.NewJson(body)
	if err != nil {
		log.Fatal(err)
	}
	token := json.Get("data").Get("accessToken").MustString()
	expireTime := json.Get("data").Get("expireTime").MustInt64()
	return token, expireTime
}

func getImg(token string, sn int) (img string, err error) {
	req, err := http.NewRequest("POST",
		"https://open.ys7.com/api/lapp/device/capture",
		strings.NewReader("accessToken="+token+"&deviceSerial="+strconv.Itoa(sn)+"&channelNo=1"))
	if err != nil {
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	json, _ := simplejson.NewJson(body)
	code := json.Get("code").MustString()
	if code == "200" {
		img = json.Get("data").Get("picUrl").MustString()
	} else {
		err = errors.New(code)
	}
	return img, err
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return
	}
	defer dst.Close()

	return io.Copy(dst, src)
}

func isFile(fileName string) bool {
	if _, err := os.Stat(fileName); err == nil {
		return true
	}
	return false
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func main() {
	aKey := flag.String("appKey", "", "appKey")
	aSecret := flag.String("appSecret", "", "appSecret")
	nowpath := flag.String("nowpath", getCurrentDirectory()+"/now.jpg", "存储图片目录")
	sn := flag.Int("sn", 0, "设备序列号")
	interval := flag.Int("interval", 1, "图片抓取间隔")
	flag.Parse()
	crawlInterval := *interval
	appKey := *aKey
	appSecret := *aSecret
	deviceSerial := *sn
	savePath := *nowpath
	imgPath := getCurrentDirectory() + "/img/"
	var token string
	var expireTime int64
	ticker := time.NewTicker(time.Minute * time.Duration(crawlInterval))
	for _ = range ticker.C {
		fmt.Println(time.Now().String() + "抓取开始")
		nTime := makeTimestamp()
		if nTime > expireTime {
			token, expireTime = getToken(appKey, appSecret)
			fmt.Println("获取Token")
		}

		imgURL, err := getImg(token, deviceSerial)
		if err != nil {
			fmt.Println(time.Now().String()+"图片抓取失败，可能因为设备关机或无法链接。 Code:", err)
		} else {
			res, _ := http.Get(imgURL)
			if PathExist(savePath) {
				os.Remove(savePath)
			}
			file, _ := os.Create(savePath)
			defer file.Close()
			io.Copy(file, res.Body)
			file.Close()

			if !isFile(imgPath) {
				os.MkdirAll(imgPath, 0777)
			}

			CopyFile(imgPath+"/"+time.Now().Format("2006-01-02 15-04-05")+".jpg", savePath)

			fmt.Println(time.Now().String() + "抓取结束")
		}

	}

}
