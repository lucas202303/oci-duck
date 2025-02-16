package functions

import (
	"DuckyClient/utils"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/example/helpers"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"gopkg.in/ini.v1"
)

const (
	IPsFilePrefix = "IPs"
)

var (
	Current_Profile_Num = -1
	configFilePath      string
	provider            common.ConfigurationProvider
	computeClient       core.ComputeClient
	networkClient       core.VirtualNetworkClient
	storageClient       core.BlockstorageClient
	identityClient      identity.IdentityClient
	ctx                 context.Context
	Oracle_Section_Row  []*ini.Section
	Oracle_Section      *ini.Section
	Oracle_Section_Name string
	oracle              Oracle
	instance            Instance
	AvailabilityDomains []identity.AvailabilityDomain
)

func Oracle_Init() {

	// 尝试解析 Conf
	flag.StringVar(&configFilePath, "config", "./conf.ini", "配置文件路径")
	flag.StringVar(&configFilePath, "c", "./conf.ini", "配置文件路径")
	flag.Parse()
	cfg, err := ini.Load(configFilePath)
	helpers.FatalIfError(err)
	rand.Seed(time.Now().UnixNano())

	// 从 Conf 内获取甲骨文配置
	sections := cfg.Sections()
	Oracle_Section_Row = []*ini.Section{}
	for _, sec := range sections {
		if len(sec.ParentKeys()) == 0 {
			user := sec.Key("user").Value()
			fingerprint := sec.Key("fingerprint").Value()
			tenancy := sec.Key("tenancy").Value()
			region := sec.Key("region").Value()
			key_file := sec.Key("key_file").Value()
			if user != "" && fingerprint != "" && tenancy != "" && region != "" && key_file != "" {
				Oracle_Section_Row = append(Oracle_Section_Row, sec)
			}
		}
	}

	// 如果无配置
	if len(Oracle_Section_Row) == 0 {
		log.Printf("[Warn] [System] 无甲骨文正确的配置信息, 请参考链接文档：https://docs.duckawa.me/start/lian-jie-ducky-client")
		time.Sleep(3 * time.Second)
		os.Exit(3)

	}

	log.Printf("[Info] [Oracle] 成功获取到 %v 个甲骨文配置", len(Oracle_Section_Row))
	if Current_Profile_Num == -1 {
		Current_Profile_Num = 1
	}
}

func Oracle_List_Profile(content *gin.Context) {

	if Current_Profile_Num == 0 {
		Current_Profile_Num = 1
	}
	ctx = context.Background()
	var name []string
	var region []string
	for i := 0; i < len(Oracle_Section_Row); i++ {
		Oracle_Section = Oracle_Section_Row[i]
		Oracle_Section_Name = Oracle_Section.Name()
		name = append(name, Oracle_Section.Name())
		region = append(region, utils.Config(Oracle_Section_Name, "region"))
	}
	data := map[string]interface{}{
		"msg":     "success",
		"code":    0,
		"sum":     len(Oracle_Section_Row),
		"region":  region,
		"profile": name,
		"current": Current_Profile_Num,
	}
	content.JSON(200, data)
}

func Oracle_Change_Profile(content *gin.Context) {
	if content.Query("id") == "" {
		data := map[string]interface{}{
			"msg":  "missing argues",
			"code": 500,
		}
		content.JSON(500, data)
	} else {
		num, _ := strconv.Atoi(content.Query("id"))
		if num <= len(Oracle_Section_Row) {
			Current_Profile_Num = num
			Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
			data := map[string]interface{}{
				"msg":  "Success",
				"code": 0,
			}
			content.JSON(200, data)
		} else {
			data := map[string]interface{}{
				"msg":  "err",
				"code": 500,
			}
			content.JSON(500, data)
		}
	}
}

func Oracle_List_AD(content *gin.Context) {
	code, data := Oracle_List_AD_Request()
	content.JSON(code, data)
}

func Oracle_List_AD_Request() (int, map[string]interface{}) {

	// 先获取账户配置
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	var err error
	ctx = context.Background()

	err = Oracle_Account_Init_Var(Oracle_Section)
	if err != nil {
		data := map[string]interface{}{
			"code": 0,
			"msg":  "failed",
			"sum":  0,
			"ad":   "",
		}
		return 200, data
	}
	ctx = context.Background()
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	Oracle_Section_Name = Oracle_Section.Name()

	// 获取可用性域
	log.Printf("[Info] [%s] 正在获取可用性域...", Oracle_Section_Name)
	AvailabilityDomains, err := Oracle_List_AD_Handle()
	if err != nil {
		log.Printf("[Warn] [%s] 获取可用性域失败: %s", Oracle_Section_Name, err.Error())
		data := map[string]interface{}{
			"code": 0,
			"msg":  "failed",
			"sum":  0,
			"ad":   "",
		}
		return 200, data
	} else {
		var AD []string
		for i := 0; i < len(AvailabilityDomains); i++ {
			adName := AvailabilityDomains[i].Name
			AD = append(AD, string(*adName))
		}
		data := map[string]interface{}{
			"code": 0,
			"msg":  "success",
			"sum":  len(AD),
			"ad":   AD,
		}
		return 200, data
	}
}

func Oracle_Instance_Lanuch(content *gin.Context) {

	// 先获取账户配置
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	var err error
	ctx = context.Background()

	err = Oracle_Account_Init_Var(Oracle_Section)
	if err != nil {
		log.Println("err")
	}
	ctx = context.Background()
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	Oracle_Section_Name = Oracle_Section.Name()

	// 获取可用性域
	AvailabilityDomains, _ = Oracle_List_AD_Handle()

	// 转化部分量
	f, _ := strconv.ParseFloat(content.Query("Core"), 32)
	Core := float32(f)
	d, _ := strconv.ParseFloat(content.Query("Ram"), 32)
	Ram := float32(d)
	Disk, _ := strconv.ParseInt(content.Query("Disk"), 10, 64)
	i, _ := strconv.ParseInt(content.Query("Sum"), 10, 32)
	Sum := int32(i)
	n, _ := strconv.ParseInt(content.Query("MinTime"), 10, 32)
	MinTime := int32(n)
	m, _ := strconv.ParseInt(content.Query("MaxTime"), 10, 32)
	MaxTime := int32(m)

	SSH_Public_Key := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDIknOJVwm3MLQQwXj2tCQjMSUNIhhbVw0QmSkMO7DnI8uGcPYgWeRZKLTUdA2V6seyUR8ARJJYbeMh75AWl7de11gRtysjXux4xtRlgt1B0c9ztzy+ctnBE2x4jY9eFtSyP58BbJytxQVduj0TfPB7ajRQ8n7wt2UeLN7/rtTzlS3paVRQ2VjvTIBxmGfXk/VDuVjkvgxbiGRfsEgvZm1a3KzEqfkZY/NM+lNS/fa0xyai9uhX1bBnIsLnE5INyQMJEY7MY1fvsHl72gEyvkfjt0JvPvzxBK05G+i5m5YV9vt73rVFpEviI302qZg+6t9nIINScYZFnwc+M/VEPjLn ssh-key-2023-02-11"
	// 结构体
	instance = Instance{
		AvailabilityDomain:       "",
		SSH_Public_Key:           SSH_Public_Key,
		Vcn_Display_Name:         "",
		Subnet_Display_Name:      "",
		Shape:                    content.Query("Shape"),
		Operating_System:         content.Query("Os"),
		Operating_System_Version: content.Query("Os_Version"),
		Instance_Display_Name:    content.Query("Name"),
		Core:                     Core,
		Ram:                      Ram,
		Disk:                     Disk,
		Sum:                      Sum,
		MinTime:                  MinTime,
		MaxTime:                  MaxTime,
	}

	go Oracle_Instance_Lanuch_Handle(AvailabilityDomains, content.Query("AD"), content.Query("Os"))
}

func Oracle_Instance_Lanuch_Handle(ADs []identity.AvailabilityDomain, AD_Auto string, Os string) {

	// 检查可用性域数量
	var AD_num int32 = int32(len(ADs))
	var Current_AD_ID = 3

	// 实例自定义名称
	name := instance.Instance_Display_Name
	if name == "Default_Name" {
		name = time.Now().Format("instance-20060102-1504")
	}

	// 创建实例的请求
	request := core.LaunchInstanceRequest{}
	request.CompartmentId = common.String(oracle.Tenancy)
	request.DisplayName = common.String(name)

	// 获取系统镜像
	log.Printf("[Info] [%s] 正在获取系统镜像...", Oracle_Section_Name)
	image, err := Oracle_Get_Image(ctx, computeClient)
	if err != nil {
		log.Printf("[Warn] [%s] 获取系统镜像失败:%s", Oracle_Section_Name, err.Error())
	} else {
		log.Printf("[Info] [%s] 系统镜像: %s", Oracle_Section_Name, *image.DisplayName)
	}

	// 获取实例型号
	var shape core.Shape
	if strings.Contains(strings.ToLower(instance.Shape), "flex") && instance.Core > 0 && instance.Ram > 0 {
		shape.Shape = &instance.Shape
		shape.Ocpus = &instance.Core
		shape.MemoryInGBs = &instance.Ram
	} else {
		log.Printf("[Info] [%s] 正在获取 Shape 信息...", Oracle_Section_Name)
		shape, err = Oracle_Get_Shape(image.Id, instance.Shape)
		if err != nil {
			log.Printf("[Warn] [%s] 获取 Shape 信息失败: %s", Oracle_Section_Name, err.Error())
			return
		}
	}
	request.Shape = shape.Shape
	var Core_New = "0"
	var Ram_New = "0"
	if strings.Contains(strings.ToLower(*shape.Shape), "flex") {
		request.ShapeConfig = &core.LaunchInstanceShapeConfigDetails{
			Ocpus:       shape.Ocpus,
			MemoryInGBs: shape.MemoryInGBs,
		}
		Core_New = strings.Replace(fmt.Sprintln(instance.Core), "\n", "", -1)
		Ram_New = strings.Replace(fmt.Sprintln(instance.Ram), "\n", "", -1)
	} else {
		Core_New = "1"
		Ram_New = "1"
	}

	// 创建子网或获取已创建的子网
	subnet, err := Oracle_Get_Network(ctx, networkClient)
	if err != nil {
		log.Printf("[Warn] [%s] 获取子网失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	log.Printf("[Info] [%s] 子网: %s", Oracle_Section_Name, *subnet.DisplayName)
	request.CreateVnicDetails = &core.CreateVnicDetails{SubnetId: subnet.Id}

	// 硬盘
	sd := core.InstanceSourceViaImageDetails{}
	sd.ImageId = image.Id
	if instance.Disk > 0 {
		sd.BootVolumeSizeInGBs = common.Int64(instance.Disk)
	}
	request.SourceDetails = sd
	request.IsPvEncryptionInTransitEnabled = common.Bool(true)

	// 公钥
	metaData := map[string]string{}
	metaData["ssh_authorized_keys"] = instance.SSH_Public_Key
	request.Metadata = metaData

	// 开始刷鸡
	/*
		1.自动AD    一个AD  --- 正常循环
		2.指定AD    一个AD  --- 正常循环
		3.自动AD    三个AD  --- 循环AD刷
		4.指定AD    三个AD  --- 正常循环
	*/

	var SumCreated = 0
	var Oracle_Lanuch_Instance_Attempts = 0
	var Three_AD_Attempts_Count = 0
	var Three_AD_Error_1 string
	var Three_AD_Error_2 string
	var Three_AD_Error_3 string

	// 如果为 1、2、4
	if AD_Auto == "AD-1" || AD_Auto == "AD-2" || AD_Auto == "AD-3" || AD_Auto == "Auto" && AD_num == 1 {
		if AD_Auto == "Auto" && AD_num == 1 {
			AD_Auto = "AD-1"
		}
		log.Printf("[Info] [%s] 正在尝试创建实例, AD: %s", Oracle_Section_Name, AD_Auto)
		// 如果为 3
	} else if AD_Auto == "Auto" && AD_num != 1 {
		log.Printf("[Info] [%s] 正在尝试创建实例, AD: Auto", Oracle_Section_Name)
	}

	// 开始刷鸡
	for {
		if SumCreated == int(instance.Sum) {
			break
		}
		start := time.Now()
		// 如果为 1、2、4
		if AD_Auto == "AD-1" || AD_Auto == "AD-2" || AD_Auto == "AD-3" || AD_Auto == "Auto" && AD_num == 1 {

			// 判断当前 AD 的 id
			if AD_Auto == "AD-1" {
				Current_AD_ID = 1
			}
			if AD_Auto == "AD-2" {
				Current_AD_ID = 2
			}
			if AD_Auto == "AD-3" {
				Current_AD_ID = 3
			}
			if AD_Auto == "Auto" && AD_num == 1 {
				Current_AD_ID = 1
			}
			// 如果为 3
		} else if AD_Auto == "Auto" && AD_num != 1 {
			// 切换 AD
			if Current_AD_ID == 1 {
				Current_AD_ID = 2
			} else if Current_AD_ID == 2 {
				Current_AD_ID = 3
			} else if Current_AD_ID == 3 {
				Current_AD_ID = 1
			}
		}

		Oracle_Lanuch_Instance_Attempts++
		request.AvailabilityDomain = ADs[Current_AD_ID-1].Name
		createResp, err := computeClient.LaunchInstance(ctx, request)

		if err == nil { // 创建成功

			// 获取实例公共IP
			var Oracle_Instance_IP string
			ips, err := Oracle_Get_Instance_Public_Ips(createResp.Instance.Id)

			// 启动失败的
			if err != nil {
				// TG 通知
				line1 := "#开机失败\n"
				line2 := "错误信息："
				line3 := err.Error()
				text := "*" + line1 + line2 + "*" + line3
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
				for {
					url := ApiBase + "notice/new" + "?user=" + utils.Config("Client", "User") + "&data=" + url.QueryEscape(data)
					req, _ := http.NewRequest("GET", url, nil)
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Printf("[Warn] [System] Api 连接出现未知错误：%s", err.Error())
					} else {
						defer res.Body.Close()
						body, _ := io.ReadAll(res.Body)
						log.Printf("[Info] [System] Api 返回 ：%s", string(body))
						break
					}
				}

				// 停止刷鸡
				break
				// 启动成功
			} else {
				Oracle_Instance_IP = strings.Join(ips, ",")
				log.Printf("[Info] [%s] 实例抢到了, 启动成功，当前尝试次数: %d，实例名称: %s, 公共IP: %s", Oracle_Section_Name, Oracle_Lanuch_Instance_Attempts, *createResp.Instance.DisplayName, Oracle_Instance_IP)
				Oracle_Instance_Key := utils.RandomkeyGenerate("abcdefghijklmnopqrstuvwxyz0123456789", 12, "true")

				// 定义
				var Oracle_Instance_AD string
				if utils.Match("*AD-1", *ADs[Current_AD_ID-1].Name) == "true" {
					Oracle_Instance_AD = "AD-1"
				} else if utils.Match("*AD-2", *ADs[Current_AD_ID-1].Name) == "true" {
					Oracle_Instance_AD = "AD-2"
				} else if utils.Match("*AD-3", *ADs[Current_AD_ID-1].Name) == "true" {
					Oracle_Instance_AD = "AD-3"
				} else {
					Oracle_Instance_AD = "Error"
				}
				var FD string
				if utils.Match("*1", *createResp.Instance.FaultDomain) == "true" {
					FD = "FD-1"
				} else if utils.Match("*2", *createResp.Instance.FaultDomain) == "true" {
					FD = "FD-2"
				} else if utils.Match("*3", *createResp.Instance.FaultDomain) == "true" {
					FD = "FD-3"
				}
				time := FormatDuration(time.Since(start))

				// TG 通知
				line1 := "———   🎉 恭喜 🎉，#开机成功    ———\n"
				line2 := "\n"
				line3 := "———————  实例配置  ———————\n"
				line4 := "Profile  : " + Oracle_Section_Name + "\n"
				line5 := "Region : " + utils.Config(Oracle_Section_Name, "region") + "\n"
				line5_1 := "Name  : " + *createResp.Instance.DisplayName + "\n"
				line5_5 := "AD        : " + Oracle_Instance_AD + "\n"
				line5_6 := "FD        : " + FD + "\n"
				line6 := "Shape  : " + instance.Shape + "\n"
				line6_1 := "Cpu      : " + *createResp.Instance.ShapeConfig.ProcessorDescription + "\n"
				line7 := "OS        : " + Os + " " + instance.Operating_System_Version + "\n"
				line8 := "Cores   : " + Core_New + "\n"
				line9 := "Ram     : " + Ram_New + " GB\n"
				line10 := "Disk      : " + strings.Replace(fmt.Sprintln(instance.Disk), "\n", "", -1) + " GB\n"
				line11 := "———————  连接信息  ———————\n"
				line12 := "IPv4     : `" + Oracle_Instance_IP + "`\n"
				line13 := "User    : `" + "root" + "`\n"
				line14 := "Pass    : `" + Oracle_Instance_Key + "`\n"
				line15 := "———————  其他信息  ———————\n"
				var line15_5 string
				if int(instance.Sum) != 1 {
					line15_5 = "开机数量：第 " + strconv.Itoa(SumCreated+1) + " 个\n"
				}
				line16 := "抢鸡次数：第 " + strings.Replace(fmt.Sprintln(Oracle_Lanuch_Instance_Attempts), "\n", "", -1) + " 次\n"
				line17 := "花费时间：" + time + "\n"
				line18 := "———————  注意信息  ———————\n"
				line19 := "1. root 密码 Ducky Bot 不会自动保存\n"
				line20 := "    请自行保存；\n"
				line21 := "2. root 密码可能需要 120s 或更长时间生效. \n"
				line22 := ". "
				text := "*" + line1 + line2 + line3 + line4 + line5 + line5_1 + line5_5 + line5_6 + line6 + line6_1 + line7 + line8 + line9 + line10 + line11 + line12 + line13 + line14 + line15 + line15_5 + line16 + line17 + line18 + line19 + line20 + line21 + line22 + "*"
				text = strings.Replace(text, "#", "\\#", -1)
				text = strings.Replace(text, "(", "\\(\\", -1)
				text = strings.Replace(text, ")", "\\)\\", -1)
				text = strings.Replace(text, ".", "\\.\\", -1)
				data := strings.Replace(text, "-", "\\-\\", -1)

				// 重置变量
				Oracle_Lanuch_Instance_Attempts = 0
				Three_AD_Attempts_Count = 0
				SumCreated++

				// 上报给 Api
				var ApiBase string
				if utils.Config("Client", "Api") == "" {
					ApiBase = "https://api.duckawa.me/api/v2/"
				} else {
					ApiBase = "https://api-beta.duckawa.me/api/v2/"
				}

				for {
					url := ApiBase + "notice/new" + "?user=" + utils.Config("Client", "User") + "&data=" + url.QueryEscape(data)
					req, _ := http.NewRequest("GET", url, nil)
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Printf("[Warn] [System] Api 连接出现未知错误：%s", err.Error())
					} else {
						defer res.Body.Close()
						_, _ = io.ReadAll(res.Body)
						break
					}
				}

				// 更改 Root 密码
				go utils.RootPassword(Oracle_Instance_IP, Oracle_Instance_Key)
			}

		} else {

			// 查错误
			servErr, isServErr := common.IsServiceError(err)

			// 如果是不可重试的错误
			if isServErr && (400 <= servErr.GetHTTPStatusCode() && servErr.GetHTTPStatusCode() <= 405) ||
				(servErr.GetHTTPStatusCode() == 409 && !strings.EqualFold(servErr.GetCode(), "IncorrectState")) ||
				servErr.GetHTTPStatusCode() == 412 || servErr.GetHTTPStatusCode() == 413 || servErr.GetHTTPStatusCode() == 422 ||
				servErr.GetHTTPStatusCode() == 431 || servErr.GetHTTPStatusCode() == 501 {

				// 不可重试
				if isServErr {
					errInfo := servErr.GetMessage()
					log.Printf("[Warn] [%s] 实例创建出现错误了, 错误信息: %s", Oracle_Section_Name, errInfo)
					var Error string
					Error = errInfo
					if AD_Auto == "Auto" && AD_num != 1 {
						Three_AD_Attempts_Count++
						Three_AD_Error_1 = errInfo
					}
					if Three_AD_Attempts_Count == 1 {
						Three_AD_Error_1 = errInfo
					} else if Three_AD_Attempts_Count == 2 {
						Three_AD_Error_2 = errInfo
					} else if Three_AD_Attempts_Count == 3 {
						Three_AD_Error_3 = errInfo
					}

					if (AD_Auto == "Auto" && AD_num != 1 && Three_AD_Attempts_Count == 3) || AD_Auto != "Auto" {

						if AD_Auto == "Auto" && AD_num != 1 && Three_AD_Attempts_Count == 3 {
							if Three_AD_Error_1 == Three_AD_Error_2 && Three_AD_Error_2 == Three_AD_Error_3 {
								Error = Three_AD_Error_1 + " "
							} else if Three_AD_Error_1 == Three_AD_Error_2 {
								Error = Three_AD_Error_3 + " "
							} else if Three_AD_Error_2 == Three_AD_Error_3 {
								Error = Three_AD_Error_1 + " "
							} else if Three_AD_Error_1 == Three_AD_Error_3 {
								Error = Three_AD_Error_2 + " "
							} else {
								Error = Three_AD_Error_2 + " "
							}
						}

						// TG 通知
						line1 := "#开机失败\n"
						line2 := "错误信息："
						line3 := Error
						text := "*" + line1 + line2 + "*" + line3
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
						for {
							url := ApiBase + "notice/new" + "?user=" + utils.Config("Client", "User") + "&data=" + url.QueryEscape(data)
							req, _ := http.NewRequest("GET", url, nil)
							res, err := http.DefaultClient.Do(req)
							if err != nil {
								log.Printf("[Warn] [System] Api 连接出现未知错误：%s", err.Error())
							} else {
								defer res.Body.Close()
								body, _ := io.ReadAll(res.Body)
								log.Printf("[Info] [System] Api 返回 ：%s", string(body))
								break
							}
						}

						// 重置数据
						Three_AD_Attempts_Count = 0

						// 停止刷鸡
						break
					}
				}

			} else {
				// 可重试
				if isServErr {
					errInfo := servErr.GetMessage()
					log.Printf("[Warn] [%s] 创建失败，当前尝试次数: %d (%s)", Oracle_Section_Name, Oracle_Lanuch_Instance_Attempts, errInfo)
					utils.SleepRandomSecond(instance.MinTime, instance.MaxTime)
				}
			}
		}
	}
}

func Oracle_Instance_List_Handle(content *gin.Context) {

	// 请求头
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	var err error
	ctx = context.Background()
	err = Oracle_Account_Init_Var(Oracle_Section)
	if err != nil {
		log.Printf("[Info] [%s] 获取失败.", Oracle_Section_Name)
	}
	log.Printf("[Info] [%s] 正在获取实例数据.", Oracle_Section_Name)
	instances, err := ListInstances(ctx, computeClient)
	if err != nil {
		log.Printf("[Info] [%s] 获取失败.", Oracle_Section_Name)
	}
	if len(instances) == 0 {
		log.Printf("[Info] [%s] 实例为空.", Oracle_Section_Name)
	}

	// 定义量
	var Name, Ip, Shape, Status, Core, Ram []string

	// 获取实例信息
	for i, ins := range instances {

		// 获取实例公共IP
		var strIps string
		ips, err := Oracle_Get_Instance_Public_Ips(instances[i].Id)
		if err != nil {
			strIps = err.Error()
		} else {
			strIps = strings.Join(ips, ",")
		}
		Name = append(Name, *ins.DisplayName)
		Ip = append(Ip, strIps)
		Shape = append(Shape, *ins.Shape)
		Status = append(Status, getInstanceState(ins.LifecycleState))
		Core = append(Core, strconv.FormatFloat(float64(*ins.ShapeConfig.Ocpus), 'f', -1, 32))
		Ram = append(Ram, strconv.FormatFloat(float64(*ins.ShapeConfig.MemoryInGBs), 'f', -1, 32))
	}

	// 返回数据
	data := map[string]interface{}{
		"msg":    "success",
		"code":   0,
		"name":   Name,
		"ip":     Ip,
		"shape":  Shape,
		"status": Status,
		"core":   Core,
		"ram":    Ram,
		"sum":    len(Name),
	}
	content.JSON(200, data)
}

func Oracle_Instance_Manage(content *gin.Context) {

	if content.Query("action") == "" || content.Query("id") == "" {
		data := map[string]interface{}{
			"msg":  "missing argues",
			"code": 500,
		}
		content.JSON(500, data)
	} else {
		num, _ := strconv.Atoi(content.Query("id"))
		code, data := Oracle_Instance_Manage_Handle(content.Query("action"), num, content.Query("text"))
		content.JSON(code, data)
	}
}

func Oracle_Instance_Manage_Handle(action string, index int, text string) (int, map[string]interface{}) {

	// 请求头
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	var err error
	ctx = context.Background()
	err = Oracle_Account_Init_Var(Oracle_Section)
	if err != nil {
		log.Printf("[Info] [%s] 实例数据获取失败.", Oracle_Section_Name)
		return 500, map[string]interface{}{
			"msg":  err,
			"code": 500,
			"text": "",
		}
	}
	instances, err := ListInstances(ctx, computeClient)
	if err != nil {
		log.Printf("[Info] [%s] 实例数据获取失败.", Oracle_Section_Name)
		return 500, map[string]interface{}{
			"msg":  err,
			"code": 500,
			"text": "",
		}
	}
	if len(instances) == 0 {
		return 500, map[string]interface{}{
			"msg":  "blank",
			"code": 500,
			"text": "",
		}
	}
	if index > len(instances) {
		return 500, map[string]interface{}{
			"msg":  "blank",
			"code": 500,
			"text": "",
		}
	}

	// 开始处理
	if action == "start" {
		_, err = instanceAction(instances[index-1].Id, core.InstanceActionActionStart)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 启动失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 启动成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "stop_soft" {
		_, err = instanceAction(instances[index-1].Id, core.InstanceActionActionSoftstop)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 关机失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 关机成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "stop_force" {
		_, err = instanceAction(instances[index-1].Id, core.InstanceActionActionStop)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 强制关机失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 强制关机成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "restart_soft" {
		_, err = instanceAction(instances[index-1].Id, core.InstanceActionActionSoftreset)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 重启失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 重启成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "restart_force" {
		_, err = instanceAction(instances[index-1].Id, core.InstanceActionActionReset)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 强制重启失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 强制重启成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "terminate" {
		err := terminateInstance(instances[index-1].Id)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 终止失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 终止成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "rename" {
		_, err := Oracle_Instance_Rename(instances[index-1].Id, text)
		if err != nil {
			log.Printf("[Info] [%s] 实例 %s 重命名失败:%s", Oracle_Section_Name, *instances[index-1].DisplayName, err.Error())
			return 500, map[string]interface{}{
				"msg":  err.Error(),
				"code": 500,
				"text": "",
			}
		} else {
			log.Printf("[Info] [%s] 实例 %s 重命名成功", Oracle_Section_Name, *instances[index-1].DisplayName)
			return 200, map[string]interface{}{
				"msg":  "success",
				"code": 0,
				"text": "",
			}
		}
	} else if action == "detail" {
		data := Oracle_Instance_Detail(instances[index-1].Id)
		log.Printf("[Info] [%s] 实例 %s 详细信息", Oracle_Section_Name, *instances[index-1].DisplayName)
		return 200, map[string]interface{}{
			"msg":  "success",
			"code": 0,
			"text": data,
		}

	} else {
		return 500, map[string]interface{}{
			"msg":  "failed",
			"code": 500,
			"text": "",
		}
	}
}

func Oracle_Demo(Email string) {

	// 请求头
	Oracle_Section = Oracle_Section_Row[Current_Profile_Num-1]
	var err error
	ctx = context.Background()
	err = Oracle_Account_Init_Var(Oracle_Section)
	if err != nil {
		log.Printf("[Info] [%s] 用户数据获取失败.", Oracle_Section_Name)
	}

	// 创建一个新的用户请求
	Oracle_User_Create_Request := identity.CreateUserRequest{
		CreateUserDetails: identity.CreateUserDetails{
			Name:        common.String(Email),
			Email:       common.String(Email),
			Description: common.String(Email),
		},
	}
	Oracle_User_Create_Request.CompartmentId = common.String(oracle.Tenancy)

	// 调用身份管理 API 创建用户
	Oracle_User_Create_Result, err := identityClient.CreateUser(ctx, Oracle_User_Create_Request)
	if err != nil {
		log.Printf("[Info] [%s] 用户 %s 创建失败:%s", Oracle_Section_Name,Email,err.Error())
	}
	log.Printf("[Info] [%s] 用户 %s 创建成功.", Oracle_Section_Name,Email)

	// 列取 Group
	Oracle_Group_List_Request := identity.ListGroupsRequest{
		CompartmentId: common.String(oracle.Tenancy),
	}

	// 发送请求
	Oracle_Group_List_Result, err := identityClient.ListGroups(context.Background(), Oracle_Group_List_Request)
	if err != nil {
		log.Printf("[Info] [%s] 列取组失败:%s", Oracle_Section_Name,err.Error())
	}

	// ID
	for _, group := range Oracle_Group_List_Result.Items {
		log.Printf("[Info] [%s] 用户 %s 正在添加入组 %s ...", Oracle_Section_Name,Email,*group.Name)
		time.Sleep(5 * time.Second)
		request := identity.AddUserToGroupRequest{
			AddUserToGroupDetails: identity.AddUserToGroupDetails{
				GroupId: common.String(*group.Id),
				UserId:  common.String(*Oracle_User_Create_Result.User.Id),
			},
		}

		// 发送请求
		_, err := identityClient.AddUserToGroup(context.Background(), request)
		if err != nil {
			log.Printf("[Info] [%s] 为用户 %s 添加入组 %s 失败：%s", Oracle_Section_Name,Email,*group.Name,err.Error())
		}

	}
	log.Printf("[Info] [%s] 用户 %s 创建成功且已经添加所有组.", Oracle_Section_Name,Email)
}

/*
   ###############################
   ###### 下面是杂项（无需动）######
   ###############################
*/
func Oracle_Account_Init_Var(Oracle_Section *ini.Section) (err error) {
	Oracle_Section_Name = Oracle_Section.Name()
	oracle = Oracle{}
	err = Oracle_Section.MapTo(&oracle)
	if err != nil {
		log.Printf("[Warn] [%s] 解析账号相关参数失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	provider, err = Oracle_Get_Provider(oracle)
	if err != nil {
		log.Printf("[Warn] [%s] 获取 Provider 失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	computeClient, err = core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		log.Printf("[Warn] [%s] 创建 ComputeClient 失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	networkClient, err = core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		log.Printf("[Warn] [%s] 创建 VirtualNetworkClient 失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	storageClient, err = core.NewBlockstorageClientWithConfigurationProvider(provider)
	if err != nil {
		log.Printf("[Warn] [%s] 创建 BlockstorageClient 失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	identityClient, err = identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		log.Printf("[Warn] [%s] 创建 IdentityClient 失败: %s", Oracle_Section_Name, err.Error())
		return
	}
	return
}

func Oracle_Get_Provider(oracle Oracle) (common.ConfigurationProvider, error) {
	content, err := os.ReadFile(oracle.Key_file)
	if err != nil {
		return nil, err
	}
	privateKey := string(content)
	privateKeyPassphrase := common.String(oracle.Key_password)
	return common.NewRawConfigurationProvider(oracle.Tenancy, oracle.User, oracle.Region, oracle.Fingerprint, privateKey, privateKeyPassphrase), nil
}

// 列出符合条件的可用性域
func Oracle_List_AD_Handle() ([]identity.AvailabilityDomain, error) {
	req := identity.ListAvailabilityDomainsRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := identityClient.ListAvailabilityDomains(ctx, req)
	return resp.Items, err
}

func getCustomRequestMetadataWithRetryPolicy() common.RequestMetadata {
	return common.RequestMetadata{
		RetryPolicy: getCustomRetryPolicy(),
	}
}

func getCustomRetryPolicy() *common.RetryPolicy {
	attempts := uint(3)
	retryOnAllNon200ResponseCodes := func(r common.OCIOperationResponse) bool {
		return !(r.Error == nil && 199 < r.Response.HTTPResponse().StatusCode && r.Response.HTTPResponse().StatusCode < 300)
	}
	policy := common.NewRetryPolicyWithOptions(
		common.WithConditionalOption(!false, common.ReplaceWithValuesFromRetryPolicy(common.DefaultRetryPolicyWithoutEventualConsistency())),
		common.WithMaximumNumberAttempts(attempts),
		common.WithShouldRetryOperation(retryOnAllNon200ResponseCodes))
	return &policy
}

// 获取符合条件系统镜像中的第一个
func Oracle_Get_Image(ctx context.Context, c core.ComputeClient) (image core.Image, err error) {
	var images []core.Image
	images, err = Oracle_Get_Image_Handle(ctx, c)
	if err != nil {
		return
	}
	if len(images) > 0 {
		image = images[0]
	} else {
		err = fmt.Errorf("未找到[%s %s]的镜像, 或该镜像不支持[%s]", instance.Operating_System, instance.Operating_System_Version, instance.Shape)
	}
	return
}

func Oracle_Get_Shape(imageId *string, shapeName string) (core.Shape, error) {
	var shape core.Shape
	shapes, err := Oracle_Get_Shape_Handle(ctx, computeClient, imageId)
	if err != nil {
		return shape, err
	}
	for _, s := range shapes {
		if strings.EqualFold(*s.Shape, shapeName) {
			shape = s
			return shape, nil
		}
	}
	err = errors.New("没有符合条件的Shape")
	return shape, err
}

// 列出所有符合条件的系统镜像
func Oracle_Get_Image_Handle(ctx context.Context, c core.ComputeClient) ([]core.Image, error) {
	if instance.Operating_System == "" || instance.Operating_System_Version == "" {
		return nil, errors.New("操作系统类型和版本不能为空, 请检查配置文件")
	}
	request := core.ListImagesRequest{
		CompartmentId:          common.String(oracle.Tenancy),
		OperatingSystem:        common.String(instance.Operating_System),
		OperatingSystemVersion: common.String(instance.Operating_System_Version),
		Shape:                  common.String(instance.Shape),
		RequestMetadata:        getCustomRequestMetadataWithRetryPolicy(),
	}
	r, err := c.ListImages(ctx, request)
	return r.Items, err
}

func Oracle_Get_Shape_Handle(ctx context.Context, c core.ComputeClient, imageID *string) ([]core.Shape, error) {
	request := core.ListShapesRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		ImageId:         imageID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	r, err := c.ListShapes(ctx, request)
	if err == nil && (r.Items == nil || len(r.Items) == 0) {
		err = errors.New("没有符合条件的Shape")
	}
	return r.Items, err
}

func Oracle_Get_Network(ctx context.Context, c core.VirtualNetworkClient) (subnet core.Subnet, err error) {
	var vcn core.Vcn
	vcn, err = Oracle_Get_Vcn(ctx, c)
	if err != nil {
		return
	}
	var gateway core.InternetGateway
	gateway, err = Oracle_Get_InternetGateway(c, vcn.Id)
	if err != nil {
		return
	}
	_, err = Oracle_Get_RouteTable(c, gateway.Id, vcn.Id)
	if err != nil {
		return
	}
	subnet, err = Oracle_Get_SubnetWithDetails(
		ctx, c, vcn.Id,
		common.String(instance.Subnet_Display_Name),
		common.String("10.0.0.0/24"),
		common.String("subnetdns"),
		common.String(instance.AvailabilityDomain))
	return
}

// 创建一个新的虚拟云网络 (VCN) 或获取已经存在的虚拟云网络
func Oracle_Get_Vcn(ctx context.Context, c core.VirtualNetworkClient) (core.Vcn, error) {
	var vcn core.Vcn
	vcnItems, err := Oracle_Get_Vcn_Handle(ctx, c)
	if err != nil {
		return vcn, err
	}
	displayName := common.String(instance.Vcn_Display_Name)
	if len(vcnItems) > 0 && *displayName == "" {
		vcn = vcnItems[0]
		return vcn, err
	}
	for _, element := range vcnItems {
		if *element.DisplayName == instance.Vcn_Display_Name {
			// VCN already created, return it
			vcn = element
			return vcn, err
		}
	}
	// create a new VCN
	log.Printf("开始创建VCN（没有可用的VCN，或指定的VCN不存在）\n")
	if *displayName == "" {
		displayName = common.String(time.Now().Format("vcn-20060102-1504"))
	}
	request := core.CreateVcnRequest{}
	request.RequestMetadata = getCustomRequestMetadataWithRetryPolicy()
	request.CidrBlock = common.String("10.0.0.0/16")
	request.CompartmentId = common.String(oracle.Tenancy)
	request.DisplayName = displayName
	request.DnsLabel = common.String("vcndns")
	r, err := c.CreateVcn(ctx, request)
	if err != nil {
		return vcn, err
	}
	log.Printf("VCN创建成功: %s\n", *r.Vcn.DisplayName)
	vcn = r.Vcn
	return vcn, err
}

// 列出所有虚拟云网络 (VCN)
func Oracle_Get_Vcn_Handle(ctx context.Context, c core.VirtualNetworkClient) ([]core.Vcn, error) {
	request := core.ListVcnsRequest{
		CompartmentId:   &oracle.Tenancy,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	r, err := c.ListVcns(ctx, request)
	if err != nil {
		return nil, err
	}
	return r.Items, err
}

// 创建或者获取 Internet 网关
func Oracle_Get_InternetGateway(c core.VirtualNetworkClient, vcnID *string) (core.InternetGateway, error) {
	//List Gateways
	var gateway core.InternetGateway
	listGWRequest := core.ListInternetGatewaysRequest{
		CompartmentId:   &oracle.Tenancy,
		VcnId:           vcnID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}

	listGWRespone, err := c.ListInternetGateways(ctx, listGWRequest)
	if err != nil {
		log.Printf("Internet gateway list error: %s\n", err.Error())
		return gateway, err
	}

	if len(listGWRespone.Items) >= 1 {
		//Gateway with name already exists
		gateway = listGWRespone.Items[0]
	} else {
		//Create new Gateway
		log.Printf("开始创建Internet网关\n")
		enabled := true
		createGWDetails := core.CreateInternetGatewayDetails{
			CompartmentId: &oracle.Tenancy,
			IsEnabled:     &enabled,
			VcnId:         vcnID,
		}

		createGWRequest := core.CreateInternetGatewayRequest{
			CreateInternetGatewayDetails: createGWDetails,
			RequestMetadata:              getCustomRequestMetadataWithRetryPolicy()}

		createGWResponse, err := c.CreateInternetGateway(ctx, createGWRequest)

		if err != nil {
			log.Printf("Internet gateway create error: %s\n", err.Error())
			return gateway, err
		}
		gateway = createGWResponse.InternetGateway
		log.Printf("Internet网关创建成功: %s\n", *gateway.DisplayName)
	}
	return gateway, err
}

// 创建或者获取路由表
func Oracle_Get_RouteTable(c core.VirtualNetworkClient, gatewayID, VcnID *string) (routeTable core.RouteTable, err error) {
	//List Route Table
	listRTRequest := core.ListRouteTablesRequest{
		CompartmentId:   &oracle.Tenancy,
		VcnId:           VcnID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	var listRTResponse core.ListRouteTablesResponse
	listRTResponse, err = c.ListRouteTables(ctx, listRTRequest)
	if err != nil {
		log.Printf("Route table list error: %s\n", err.Error())
		return
	}

	cidrRange := "0.0.0.0/0"
	rr := core.RouteRule{
		NetworkEntityId: gatewayID,
		Destination:     &cidrRange,
		DestinationType: core.RouteRuleDestinationTypeCidrBlock,
	}

	if len(listRTResponse.Items) >= 1 {
		//Default Route Table found and has at least 1 route rule
		if len(listRTResponse.Items[0].RouteRules) >= 1 {
			routeTable = listRTResponse.Items[0]
			//Default Route table needs route rule adding
		} else {
			log.Printf("路由表未添加规则，开始添加Internet路由规则\n")
			updateRTDetails := core.UpdateRouteTableDetails{
				RouteRules: []core.RouteRule{rr},
			}
			updateRTRequest := core.UpdateRouteTableRequest{
				RtId:                    listRTResponse.Items[0].Id,
				UpdateRouteTableDetails: updateRTDetails,
				RequestMetadata:         getCustomRequestMetadataWithRetryPolicy(),
			}
			var updateRTResponse core.UpdateRouteTableResponse
			updateRTResponse, err = c.UpdateRouteTable(ctx, updateRTRequest)
			if err != nil {
				log.Printf("Error updating route table: %s\n", err)
				return
			}
			log.Printf("Internet路由规则添加成功\n")
			routeTable = updateRTResponse.RouteTable
		}

	} else {
		//No default route table found
		log.Printf("Error could not find VCN default route table, VCN OCID: %s Could not find route table.\n", *VcnID)
	}
	return
}

func Oracle_Get_SubnetWithDetails(ctx context.Context, c core.VirtualNetworkClient, vcnID *string,
	displayName *string, cidrBlock *string, dnsLabel *string, availableDomain *string) (subnet core.Subnet, err error) {
	var subnets []core.Subnet
	subnets, err = Oracle_Get_Subnet(ctx, c, vcnID)
	if err != nil {
		return
	}

	if displayName == nil {
		displayName = common.String(instance.Subnet_Display_Name)
	}

	if len(subnets) > 0 && *displayName == "" {
		subnet = subnets[0]
		return
	}

	// check if the subnet has already been created
	for _, element := range subnets {
		if *element.DisplayName == *displayName {
			// find the subnet, return it
			subnet = element
			return
		}
	}

	// create a new subnet
	log.Printf("开始创建Subnet（没有可用的Subnet，或指定的Subnet不存在）\n")
	// 子网名称为空，以当前时间为名称创建子网
	if *displayName == "" {
		displayName = common.String(time.Now().Format("subnet-20060102-1504"))
	}
	request := core.CreateSubnetRequest{}
	//request.AvailabilityDomain = availableDomain //省略此属性创建区域性子网(regional subnet)，提供此属性创建特定于可用性域的子网。建议创建区域性子网。
	request.CompartmentId = &oracle.Tenancy
	request.CidrBlock = cidrBlock
	request.DisplayName = displayName
	request.DnsLabel = dnsLabel
	request.RequestMetadata = getCustomRequestMetadataWithRetryPolicy()

	request.VcnId = vcnID
	var r core.CreateSubnetResponse
	r, err = c.CreateSubnet(ctx, request)
	if err != nil {
		return
	}
	// retry condition check, stop unitl return true
	pollUntilAvailable := func(r common.OCIOperationResponse) bool {
		if converted, ok := r.Response.(core.GetSubnetResponse); ok {
			return converted.LifecycleState != core.SubnetLifecycleStateAvailable
		}
		return true
	}

	pollGetRequest := core.GetSubnetRequest{
		SubnetId:        r.Id,
		RequestMetadata: helpers.GetRequestMetadataWithCustomizedRetryPolicy(pollUntilAvailable),
	}

	// wait for lifecyle become running
	_, err = c.GetSubnet(ctx, pollGetRequest)
	if err != nil {
		return
	}

	// update the security rules
	getReq := core.GetSecurityListRequest{
		SecurityListId:  common.String(r.SecurityListIds[0]),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}

	var getResp core.GetSecurityListResponse
	getResp, err = c.GetSecurityList(ctx, getReq)
	if err != nil {
		return
	}

	newRules := append(getResp.IngressSecurityRules, core.IngressSecurityRule{
		//Protocol: common.String("6"), // TCP
		Protocol: common.String("all"), // 允许所有协议
		Source:   common.String("0.0.0.0/0"),
		/*TcpOptions: &core.TcpOptions{
			DestinationPortRange: &portRange, // 省略该参数，允许所有目标端口。
		},*/
	})

	updateReq := core.UpdateSecurityListRequest{
		SecurityListId:  common.String(r.SecurityListIds[0]),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}

	updateReq.IngressSecurityRules = newRules

	_, err = c.UpdateSecurityList(ctx, updateReq)
	if err != nil {
		return
	}
	log.Printf("Subnet创建成功: %s\n", *r.Subnet.DisplayName)
	subnet = r.Subnet
	return
}

// 列出指定虚拟云网络 (VCN) 中的所有子网
func Oracle_Get_Subnet(ctx context.Context, c core.VirtualNetworkClient, vcnID *string) (subnets []core.Subnet, err error) {
	request := core.ListSubnetsRequest{
		CompartmentId:   &oracle.Tenancy,
		VcnId:           vcnID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	var r core.ListSubnetsResponse
	r, err = c.ListSubnets(ctx, request)
	if err != nil {
		return
	}
	subnets = r.Items
	return
}

// 根据实例 OCID 获取公共IP
func Oracle_Get_Instance_Public_Ips(instanceId *string) (ips []string, err error) {
	// 多次尝试，避免刚抢购到实例，实例正在预配获取不到公共IP。
	var ins core.Instance
	for i := 0; i < 100; i++ {
		if ins.LifecycleState != core.InstanceLifecycleStateRunning {
			ins, err = Oracle_Get_Instance(instanceId)
			if err != nil {
				continue
			}
			if ins.LifecycleState == core.InstanceLifecycleStateTerminating || ins.LifecycleState == core.InstanceLifecycleStateTerminated {
				err = errors.New("实例已终止😔")
				return
			}
		}

		var vnicAttachments []core.VnicAttachment
		vnicAttachments, err = Oracle_List_Vnic_Attachments(ctx, computeClient, instanceId)
		if err != nil {
			continue
		}
		if len(vnicAttachments) > 0 {
			for _, vnicAttachment := range vnicAttachments {
				vnic, vnicErr := Oracle_Get_Vnic(ctx, networkClient, vnicAttachment.VnicId)
				if vnicErr != nil {
					log.Printf("Oracle_Get_Vnic error: %s\n", vnicErr.Error())
					continue
				}
				if vnic.PublicIp != nil && *vnic.PublicIp != "" {
					ips = append(ips, *vnic.PublicIp)
				}
			}
			return
		}
		time.Sleep(3 * time.Second)
	}
	return
}

func Oracle_Get_Instance(instanceId *string) (core.Instance, error) {
	req := core.GetInstanceRequest{
		InstanceId:      instanceId,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := computeClient.GetInstance(ctx, req)
	return resp.Instance, err
}

func Oracle_List_Vnic_Attachments(ctx context.Context, c core.ComputeClient, instanceId *string) ([]core.VnicAttachment, error) {
	req := core.ListVnicAttachmentsRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy()}
	if instanceId != nil && *instanceId != "" {
		req.InstanceId = instanceId
	}
	resp, err := c.ListVnicAttachments(ctx, req)
	return resp.Items, err
}

func Oracle_Get_Vnic(ctx context.Context, c core.VirtualNetworkClient, vnicID *string) (core.Vnic, error) {
	req := core.GetVnicRequest{
		VnicId:          vnicID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := c.GetVnic(ctx, req)
	if err != nil && resp.RawResponse != nil {
		err = errors.New(resp.RawResponse.Status)
	}
	return resp.Vnic, err
}

func ListInstances(ctx context.Context, c core.ComputeClient) ([]core.Instance, error) {
	req := core.ListInstancesRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := c.ListInstances(ctx, req)
	return resp.Items, err
}

func getInstanceState(state core.InstanceLifecycleStateEnum) string {
	var friendlyState string
	switch state {
	case core.InstanceLifecycleStateMoving:
		friendlyState = "正在移动"
	case core.InstanceLifecycleStateProvisioning:
		friendlyState = "正在预配"
	case core.InstanceLifecycleStateRunning:
		friendlyState = "正在运行"
	case core.InstanceLifecycleStateStarting:
		friendlyState = "正在启动"
	case core.InstanceLifecycleStateStopping:
		friendlyState = "正在停止"
	case core.InstanceLifecycleStateStopped:
		friendlyState = "已停止"
	case core.InstanceLifecycleStateTerminating:
		friendlyState = "正在终止"
	case core.InstanceLifecycleStateTerminated:
		friendlyState = "已终止"
	default:
		friendlyState = string(state)
	}
	return friendlyState
}

func instanceAction(instanceId *string, action core.InstanceActionActionEnum) (ins core.Instance, err error) {
	req := core.InstanceActionRequest{
		InstanceId:      instanceId,
		Action:          action,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := computeClient.InstanceAction(ctx, req)
	ins = resp.Instance
	return
}

// 终止实例
func terminateInstance(id *string) error {
	request := core.TerminateInstanceRequest{
		InstanceId:         id,
		PreserveBootVolume: common.Bool(false),
		RequestMetadata:    getCustomRequestMetadataWithRetryPolicy(),
	}
	_, err := computeClient.TerminateInstance(ctx, request)
	return err
}

func FormatDuration(duration time.Duration) string {
	days := duration / (24 * time.Hour)
	duration = duration % (24 * time.Hour)
	hours := duration / time.Hour
	duration = duration % time.Hour
	minutes := duration / time.Minute
	duration = duration % time.Minute
	seconds := duration / time.Second
	var Day, Hour, Minute, Second string
	if days != 0 {
		Day = strings.Replace(fmt.Sprintln(days), "ns\n", "", -1) + " 天 "
	}
	if hours != 0 {
		Hour = strings.Replace(fmt.Sprintln(hours), "ns\n", "", -1) + " 小时 "
	}
	if minutes != 0 {
		Minute = strings.Replace(fmt.Sprintln(minutes), "ns\n", "", -1) + " 分 "
	}
	Second = strings.Replace(fmt.Sprintln(seconds), "ns\n", "", -1) + " 秒"
	return Day + Hour + Minute + Second
}

func Oracle_Instance_Rename(instanceId *string, Name string) (ins core.Instance, err error) {
	req := core.UpdateInstanceRequest{
		InstanceId:            instanceId,
		UpdateInstanceDetails: core.UpdateInstanceDetails{DisplayName: common.String(Name)},
	}
	resp, err := computeClient.UpdateInstance(ctx, req)
	return resp.Instance, err
}

func Oracle_Instance_Detail(instanceId *string) (text string) {

	log.Println("正在获取实例详细信息...")
	instance, err := Oracle_Get_Instance(instanceId)
	if err != nil {
		fmt.Printf("\033[1;31m获取实例详细信息失败, 回车返回上一级菜单.\033[0m")
	}
	vnics, err := Oracle_Get_Instance_Vnics(instanceId)
	if err != nil {
		fmt.Printf("\033[1;31m获取实例VNIC失败, 回车返回上一级菜单.\033[0m")
	}
	var publicIps = make([]string, 0)
	var strPublicIps string
	if err != nil {
		strPublicIps = err.Error()
	} else {
		for _, vnic := range vnics {
			if vnic.PublicIp != nil {
				publicIps = append(publicIps, *vnic.PublicIp)
			}
		}
		strPublicIps = strings.Join(publicIps, ",")
	}

	// TG 通知
	line1 := "#实例信息\n"
	line2 := "Profile  : " + Oracle_Section_Name + "\n"
	line3 := "Region : " + utils.Config(Oracle_Section_Name, "region") + "\n"
	line4 := "Name  : " + *instance.DisplayName + "\n"
	line5 := "AD        : " + *instance.AvailabilityDomain + "\n"
	line6 := "FD        : " + *instance.FaultDomain + "\n"
	line7 := "Shape:" + *instance.Shape + "\n"
	line8 := "Cpu:" + *instance.ShapeConfig.ProcessorDescription + "\n"
	line9 := "Cores   : " + strings.Replace(fmt.Sprintln(*instance.ShapeConfig.Ocpus), "\n", "", -1) + "\n"
	line10 := "Ram     : " + strings.Replace(fmt.Sprintln(*instance.ShapeConfig.MemoryInGBs), "\n", "", -1) + " GB\n"
	line11 := "Disk      : " + strings.Replace(fmt.Sprintln(instance.ShapeConfig.LocalDisksTotalSizeInGBs), "\n", "", -1) + " GB\n"
	line12 := "Bandwidth: " + strings.Replace(fmt.Sprintln(*instance.ShapeConfig.NetworkingBandwidthInGbps), "\n", "", -1) + "\n"
	line13 := "IPv4     : `" + strPublicIps + "`\n"
	Text := "*" + line1 + line2 + line3 + line4 + line5 + line6 + line7 + line8 + line9 + line10 + line11 + line12 + line13 + "*"
	text = strings.Replace(Text, "#", "\\#", -1)
	text = strings.Replace(text, "(", "\\(\\", -1)
	text = strings.Replace(text, ")", "\\)\\", -1)
	text = strings.Replace(text, ".", "\\.\\", -1)
	data := strings.Replace(text, "-", "\\-\\", -1)
	return data

}
func Oracle_Get_Instance_Vnics(instanceId *string) (vnics []core.Vnic, err error) {
	vnicAttachments, err := Oracle_List_Vnic_Attachments(ctx, computeClient, instanceId)
	if err != nil {
		return
	}
	for _, vnicAttachment := range vnicAttachments {
		vnic, vnicErr := Oracle_Get_Instance_Vnics_Handle(ctx, networkClient, vnicAttachment.VnicId)
		if vnicErr != nil {
			log.Printf("GetVnic error: %s\n", vnicErr.Error())
			continue
		}
		vnics = append(vnics, vnic)
	}
	return
}
func Oracle_Get_Instance_Vnics_Handle(ctx context.Context, c core.VirtualNetworkClient, vnicID *string) (core.Vnic, error) {
	req := core.GetVnicRequest{
		VnicId:          vnicID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := c.GetVnic(ctx, req)
	if err != nil && resp.RawResponse != nil {
		err = errors.New(resp.RawResponse.Status)
	}
	return resp.Vnic, err
}

type Oracle struct {
	User         string `ini:"user"`
	Fingerprint  string `ini:"fingerprint"`
	Tenancy      string `ini:"tenancy"`
	Region       string `ini:"region"`
	Key_file     string `ini:"key_file"`
	Key_password string `ini:"key_password"`
}

type Instance struct {
	AvailabilityDomain       string  `ini:"availabilityDomain"`
	SSH_Public_Key           string  `ini:"ssh_authorized_key"`
	Vcn_Display_Name         string  `ini:"vcnDisplayName"`
	Subnet_Display_Name      string  `ini:"subnetDisplayName"`
	Shape                    string  `ini:"shape"`
	Operating_System         string  `ini:"OperatingSystem"`
	Operating_System_Version string  `ini:"OperatingSystemVersion"`
	Instance_Display_Name    string  `ini:"instanceDisplayName"`
	Core                     float32 `ini:"cpus"`
	Ram                      float32 `ini:"memoryInGBs"`
	Disk                     int64   `ini:"bootVolumeSizeInGBs"`
	Sum                      int32   `ini:"sum"`
	MinTime                  int32   `ini:"minTime"`
	MaxTime                  int32   `ini:"maxTime"`
}
