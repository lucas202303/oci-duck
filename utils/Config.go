package utils

import (
	"log"
	"os"
	"time"

	DuckyBotConfig "gopkg.in/ini.v1"
)

func Config(section, key string) string {

	file, err := DuckyBotConfig.Load("./conf.ini")
	if err != nil {
		log.Printf("[WARN] [System] 配置文件读取错误，请正确的放置 Conf 文件，并以先对路径的方式启动:%s", err)
		time.Sleep(3 * time.Second)
		os.Exit(int(3))
	}
	return file.Section(section).Key(key).String()
}
