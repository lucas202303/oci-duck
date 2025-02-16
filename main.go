package main

import (
	"DuckyClient/functions"
	"DuckyClient/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 版本信息
const (
	CilentVersion    = "1.0.0-Beta-4"
	ClientNum        = "1"
	CilentUpdateTime = "2023.02.21"
)

// 开始运行
func main() {

	// 字符画
	fmt.Printf(`
   ____             _             ____                _
  |  _ \ _   _  ___| | ___   _   / ___| (_) ___ _ __ | |_ 
  | | | | | | |/ __| |/ / | | | | |   | | |/ _ \ '_ \| __|
  | |_| | |_| | (__|   <| |_| | | |___| | |  __/ | | | |_ 
  |____/ \__,_|\___|_|\_\\__, |  \____|_|_|\___|_| |_|\__|
                         |___/          

        Version: %s   UpdateTime: %s

  =========================================================
 
`, CilentVersion, CilentUpdateTime)

	// 创建一个文件句柄，打开或创建文件
	file, err := os.OpenFile("DuckyClient.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	// 设置 log 包的输出为同时写入文件和终端
	log.SetOutput(io.MultiWriter(os.Stdout, file))

	// 关闭文件句柄
	defer file.Close()

	// 先判断 Api 的根
	var ApiBase string
	if utils.Config("Client", "Api") == "" {
		ApiBase = "https://api.duckawa.me/api/v2/"
	} else {
		ApiBase = "https://api-beta.duckawa.me/api/v2/"
	}

	// 测试 Api 联通性
	log.Printf("[Info] [System] Api 连接中 ...")
	url := ApiBase + "ping"
	req, _ := http.NewRequest("GET", url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Warn] [System] Api 连接失败，请上报")
		time.Sleep(3 * time.Second)
		os.Exit(int(3))
	} else {
		defer res.Body.Close()
		type Api_Connection_Test struct {
			Code       int    `json:"code"`
			Msg        string `json:"msg"`
			UpdateTime string `json:"update_time"`
			Version    string `json:"version"`
		}
		var JsonData Api_Connection_Test
		body, _ := io.ReadAll(res.Body)
		if Error := json.Unmarshal(body, &JsonData); Error == nil {
			if JsonData.Code == 0 {
				log.Printf("[Info] [System] Api 连接成功")

			} else {
				log.Printf("[Warn] [System] Api 连接错误，请上报")
				time.Sleep(3 * time.Second)
				os.Exit(int(3))
			}
		} else {
			log.Printf("[Warn] [System] Api 连接错误，请上报")
			time.Sleep(3 * time.Second)
			os.Exit(int(3))
		}
	}

	// 检查 Client 是否过时
	go Check_Client_Outdate()
	time.Sleep(2 * time.Second)

	// 从配置中获取 Ip 和 Port
	var Client_IP, Client_Port string
	if utils.Config("Client", "Ip") == "" {
		log.Printf("[Info] [System] 未从 Conf 内指定 Ip，正在获取 Ip ...")
		url := ApiBase + "ip"
		req, _ := http.NewRequest("GET", url, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[Warn] [System] Api 连接出现未知错误")
		} else {
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)
			log.Printf("[Info] [System] 当前客户端 IP ：%s", string(body))
			Client_IP = string(body)
		}
	} else {
		log.Printf("[Info] [System] 成功从 Conf 内获取到 Ip : %s", utils.Config("Client", "Ip"))
		Client_IP = utils.Config("Client", "Ip")
	}
	if utils.Config("Client", "Port") == "" {
		log.Printf("[Info] [System] 未从 Conf 内指定 Port，默认：8088")
		Client_Port = "8088"
	} else {
		log.Printf("[Info] [System] 成功从 Conf 内获取到 Port : %s", utils.Config("Client", "Port"))
		Client_Port = utils.Config("Client", "Port")
	}

	// 检查 User 或 Key
	go ClientAuth(Client_IP, Client_Port)

	// 开启 Gin
	functions.Oracle_Init()
	time.Sleep(2 * time.Second)
	log.Printf("[Info] [System] 可以操作 Ducky Bot 了，Ducky Client 已经正确启动")

	// 关闭日志
	gin.SetMode(gin.ReleaseMode)
	if utils.Config("Client", "Debug") != "true" {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		gin.DefaultWriter = io.Discard
	}

	DuckyApi := gin.Default()
	DuckyApi.GET("/api/v1/ping", utils.ApiInfo)
	DuckyApi.GET("/api/v1/client/profile/list", functions.Oracle_List_Profile)
	DuckyApi.GET("/api/v1/client/profile/change", functions.Oracle_Change_Profile)
	DuckyApi.GET("/api/v1/oracle/instance/launch", functions.Oracle_Instance_Lanuch)
	DuckyApi.GET("/api/v1/oracle/ad/list", functions.Oracle_List_AD)
	DuckyApi.GET("/api/v1/oracle/instance/list", functions.Oracle_Instance_List_Handle)
	DuckyApi.GET("/api/v1/oracle/instance/manage", functions.Oracle_Instance_Manage)
	DuckyApi.Run(":" + Client_Port)

}

func ClientAuth(Client_IP, Client_Port string) {
	for {
		ClientAuth2(Client_IP, Client_Port)
	}
}

// 检查 User 和 Key 是否正确，兼并覆写 Client 地址
func ClientAuth2(Client_IP, Client_Port string) {

	// 先判断 Api 的根
	var ApiBase string
	if utils.Config("Client", "Api") == "" {
		ApiBase = "https://api.duckawa.me/api/v2/"
	} else {
		ApiBase = "https://api-beta.duckawa.me/api/v2/"
	}

	// 覆写 Client 地址
	url := ApiBase + "client/edit?user=" + utils.Config("Client", "User") + "&key=" + utils.Config("Client", "Key") + "&ip=" + Client_IP + "&port=" + Client_Port
	req, _ := http.NewRequest("GET", url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		time.Sleep(1 * time.Second)
	} else {
		defer res.Body.Close()
		_, _ = io.ReadAll(res.Body)
		if res.StatusCode == 403 || res.StatusCode == 500 {
			log.Printf("[Warn] [System] User 或 Key 不正确，请检查设置")
			time.Sleep(3 * time.Second)
			os.Exit(int(3))
		}
	}
	time.Sleep(5 * time.Second)
}

func Check_Client_Outdate() {
	for {
		Check_Client_Outdate_Handle()
	}
}

func Check_Client_Outdate_Handle() {

	// 先判断 Api 的根
	var ApiBase string
	if utils.Config("Client", "Api") == "" {
		ApiBase = "https://api.duckawa.me/api/v2/"
	} else {
		ApiBase = "https://api-beta.duckawa.me/api/v2/"
	}

	// 覆写 Client 地址
	url4 := ApiBase + "client/version?id=" + ClientNum
	req, _ := http.NewRequest("GET", url4, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		time.Sleep(1 * time.Second)
	} else {
		defer res.Body.Close()
		_, _ = io.ReadAll(res.Body)
		if res.StatusCode == 403 || res.StatusCode == 404 {
			log.Printf("[Warn] [System] 客户端版本老旧，请更新客户端")
			// TG 通知
			line1 := "#注意\n"
			line2 := "客户端版本老旧，请更新客户端！现在客户端已经退出！"
			text := "*" + line1 + line2 + "*"
			text = strings.Replace(text, "#", "\\#", -1)
			text = strings.Replace(text, "(", "\\(\\", -1)
			text = strings.Replace(text, ")", "\\)\\", -1)
			text = strings.Replace(text, ".", "\\.\\", -1)
			data := strings.Replace(text, "-", "\\-\\", -1)

			// 上报给 Api
			var ApiBase string
			if utils.Config("Client", "Api") == "" {
				ApiBase = "https://api.duckawa.me/api/v2/"
			} else {
				ApiBase = "https://api-beta.duckawa.me/api/v2/"
			}

			url2 := ApiBase + "notice/new" + "?user=" + utils.Config("Client", "User") + "&data=" + url.QueryEscape(data)
			req, _ := http.NewRequest("GET", url2, nil)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				time.Sleep(1 * time.Second)
			} else {
				defer res.Body.Close()
			}
			time.Sleep(1 * time.Second)
			os.Exit(int(3))
		}
	}
	time.Sleep(3 * time.Second)
}
