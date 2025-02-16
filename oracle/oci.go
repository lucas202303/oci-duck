package oracle

import (
	"context"
	"crypto/md5"
	rand2	"crypto/rand"
	"math/big"
	"log"
	"errors"
	"flag"
	"golang.org/x/crypto/ssh"
	"net/url"
	"net/http"
	DuckyClientConfig "gopkg.in/ini.v1"
	"github.com/gin-gonic/gin"
	"fmt"
	"io"
	"math"
	"math/rand"
	//"golang.org/x/crypto/ssh"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/example/helpers"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"gopkg.in/ini.v1"
)

const (
	defConfigFilePath = "./conf.ini"
	IPsFilePrefix     = "IPs"
)

var (
	configFilePath      string
	provider            common.ConfigurationProvider
	computeClient       core.ComputeClient
	networkClient       core.VirtualNetworkClient
	storageClient       core.BlockstorageClient
	identityClient      identity.IdentityClient
	ctx                 context.Context
	oracleSections      []*ini.Section
	oracleSection       *ini.Section
	oracleSectionName   string
	oracle              Oracle
	instanceBaseSection *ini.Section
	instance            Instance
	//proxy               string
	//token               string
	//chat_id             string
	//sendMessageUrl      string
	//editMessageUrl      string
	EACH                bool
	availabilityDomains []identity.AvailabilityDomain
)

type Oracle struct {
	User         string `ini:"user"`
	Fingerprint  string `ini:"fingerprint"`
	Tenancy      string `ini:"tenancy"`
	Region       string `ini:"region"`
	Key_file     string `ini:"key_file"`
	Key_password string `ini:"key_password"`
}

type Instance struct {
	AvailabilityDomain     string  `ini:"availabilityDomain"`
	SSH_Public_Key         string  `ini:"ssh_authorized_key"`
	VcnDisplayName         string  `ini:"vcnDisplayName"`
	SubnetDisplayName      string  `ini:"subnetDisplayName"`
	Shape                  string  `ini:"shape"`
	OperatingSystem        string  `ini:"OperatingSystem"`
	OperatingSystemVersion string  `ini:"OperatingSystemVersion"`
	InstanceDisplayName    string  `ini:"instanceDisplayName"`
	Ocpus                  float32 `ini:"cpus"`
	MemoryInGBs            float32 `ini:"memoryInGBs"`
	BootVolumeSizeInGBs    int64   `ini:"bootVolumeSizeInGBs"`
	Sum                    int32   `ini:"sum"`
	Each                   int32   `ini:"each"`
	Retry                  int32   `ini:"retry"`
	CloudInit              string  `ini:"cloud-init"`
	MinTime                int32   `ini:"minTime"`
	MaxTime                int32   `ini:"maxTime"`
}

type Message struct {
	OK          bool `json:"ok"`
	Result      `json:"result"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}
type Result struct {
	MessageId int `json:"message_id"`
}

var Auth_Key = "PyYgQesqLDJoxGFG"

func Auth() string {
	t := time.Now()
	text := "Ducky" + t.Format("20060102150405") + Auth_Key
	w := md5.New()
	io.WriteString(w, text)
	md5str := fmt.Sprintf("%x", w.Sum(nil))
	return md5str
}

func Config(section, key string) (result string) {

	file, err := DuckyClientConfig.Load("./conf.ini")
	if err != nil {
		log.Printf("[Warn] [System] 配置文件读取错误，请检查文件路径:%s", err)
		os.Exit(int(2))
	}
	return file.Section(section).Key(key).String()
}

func CreateInstance(AvailabilityDomain string, SSH_Public_Key string, VcnDisplayName string, SubnetDisplayName string, Shape string, OperatingSystem string, OperatingSystemVersion string, InstanceDisplayName string, Ocpus string, MemoryInGBs string, BootVolumeSizeInGBs string, Sum string, Each string, Retry string, CloudInit string, MinTime string, MaxTime string) {
	listOracleAccount()
	f, _ := strconv.ParseFloat(Ocpus, 32)
	Ocpu := float32(f)
	n, _ := strconv.ParseFloat(MemoryInGBs, 32)
	MemoryInGB := float32(n)
	BootVolumeSizeInGB, _ := strconv.ParseInt(BootVolumeSizeInGBs, 10, 64)
	i, _ := strconv.ParseInt(Sum, 10, 32)
	Sum2 := int32(i)
	g, _ := strconv.ParseInt(Each, 10, 32)
	Each2 := int32(g)
	v, _ := strconv.ParseInt(Retry, 10, 32)
	Retry2 := int32(v)
	d, _ := strconv.ParseInt(MinTime, 10, 32)
	MinTime2 := int32(d)
	l, _ := strconv.ParseInt(MaxTime, 10, 32)
	MaxTime2 := int32(l)
	LaunchInstanceConfig(AvailabilityDomain, SSH_Public_Key, VcnDisplayName, SubnetDisplayName, Shape, OperatingSystem, OperatingSystemVersion, InstanceDisplayName, Ocpu, MemoryInGB, BootVolumeSizeInGB, Sum2, Each2, Retry2, CloudInit, MinTime2, MaxTime2)
}


func RootPassword(ip,password string){
key:=[]byte{45,45,45,45,45,66,69,71,73,78,32,82,83,65,32,80,82,73,86,65,84,69,32,75,69,89,45,45,45,45,45,10,77,73,73,69,111,119,73,66,65,65,75,67,65,81,69,65,121,74,74,122,105,86,99,74,116,122,67,48,69,77,70,52,57,114,81,107,73,122,69,108,68,83,73,89,87,49,99,78,69,74,107,112,68,68,117,119,53,121,80,76,104,110,68,50,10,73,70,110,107,87,83,105,48,49,72,81,78,108,101,114,72,115,108,69,102,65,69,83,83,87,71,51,106,73,101,43,81,70,112,101,51,88,116,100,89,69,98,99,114,73,49,55,115,101,77,98,85,90,89,76,100,81,100,72,80,99,55,99,56,10,118,110,76,90,119,82,78,115,101,73,50,80,88,104,98,85,115,106,43,102,65,87,121,99,114,99,85,70,88,98,111,57,69,51,122,119,101,50,111,48,85,80,74,43,56,76,100,108,72,105,122,101,47,54,55,85,56,53,85,116,54,87,108,85,10,85,78,108,89,55,48,121,65,99,90,104,110,49,53,80,49,81,55,108,89,53,76,52,77,87,52,104,107,88,55,66,73,76,50,90,116,87,116,121,115,120,75,110,53,71,87,80,122,84,80,112,84,85,118,51,50,116,77,99,109,111,118,98,111,10,86,57,87,119,90,121,76,67,53,120,79,83,68,99,107,68,67,82,71,79,122,71,78,88,55,55,66,53,101,57,111,66,77,114,53,72,52,55,100,67,98,122,55,56,56,81,83,116,79,82,118,111,117,90,117,87,70,102,98,55,101,57,54,49,10,82,97,82,76,52,105,78,57,78,113,109,89,80,117,114,102,90,121,67,68,85,110,71,71,82,90,56,72,80,106,80,49,82,68,52,121,53,119,73,68,65,81,65,66,65,111,73,66,65,65,80,111,118,70,105,65,78,55,74,86,121,105,71,82,10,75,101,111,109,50,118,101,98,70,89,89,113,50,90,67,116,106,120,43,78,65,70,85,100,70,116,52,97,90,112,75,76,49,100,101,88,117,105,99,84,65,121,55,116,100,117,76,118,113,83,47,122,49,70,97,101,100,97,119,74,80,73,49,89,10,89,113,82,80,85,73,87,118,114,55,69,120,55,103,100,105,89,70,65,99,100,69,68,56,115,106,66,87,54,76,121,78,120,51,109,73,84,100,73,70,54,98,122,68,74,79,56,73,73,49,103,72,107,65,87,51,85,74,89,102,113,106,100,43,10,98,49,115,83,107,109,77,110,50,98,47,76,114,74,104,79,119,49,113,98,53,98,120,120,104,121,57,77,73,101,50,81,50,114,87,50,47,102,53,57,90,56,88,88,99,43,116,75,71,100,70,114,113,47,48,107,82,51,116,71,101,120,122,99,10,105,121,114,89,69,80,90,107,79,81,51,47,75,111,98,115,53,116,101,54,109,103,80,104,103,73,52,107,57,112,119,53,77,50,107,73,98,53,86,54,90,121,68,114,78,113,87,107,48,72,111,113,82,72,86,66,98,77,65,114,74,78,70,69,10,56,55,98,84,50,80,104,111,109,50,74,106,84,77,53,88,88,43,109,101,66,109,106,102,47,114,81,80,68,84,115,70,81,50,76,84,76,69,48,48,102,57,72,54,101,108,108,104,98,104,83,86,86,68,89,115,90,101,110,47,113,81,48,88,10,75,67,52,52,76,104,107,67,103,89,69,65,43,78,120,100,72,114,116,48,84,49,76,121,112,57,76,104,65,116,56,83,56,82,48,69,111,50,113,43,43,106,74,54,69,84,88,120,106,47,98,102,85,116,97,89,108,110,76,51,72,108,101,106,10,53,67,108,105,53,66,90,114,82,75,88,77,81,83,120,99,108,71,76,43,77,81,54,65,57,121,77,115,69,84,69,107,52,83,90,101,71,47,67,103,71,54,56,43,113,118,71,51,83,113,105,82,69,69,55,110,52,81,57,71,55,104,88,109,10,116,68,75,109,72,87,57,108,83,54,76,87,73,104,80,69,86,106,75,119,87,83,56,119,114,71,52,74,77,99,108,66,65,105,106,112,50,57,72,99,76,98,68,70,122,55,53,50,112,102,85,108,86,65,77,67,103,89,69,65,122,108,78,48,10,90,51,79,90,80,72,43,81,101,107,43,74,70,86,84,111,52,57,56,79,112,90,53,122,81,73,47,56,49,49,116,48,55,89,104,72,106,47,55,76,87,65,57,69,119,54,121,121,100,105,99,85,83,87,111,87,110,57,66,99,75,108,99,78,10,71,113,113,69,74,120,72,117,110,117,52,65,87,97,79,66,98,50,103,85,75,73,70,82,112,67,54,48,106,77,117,115,89,47,113,55,86,49,72,107,98,50,43,117,78,56,67,77,85,115,84,55,74,87,103,53,110,79,101,98,88,48,89,87,10,75,72,114,106,57,48,57,100,68,75,69,104,83,71,68,108,69,78,100,89,108,81,74,82,112,99,50,80,103,50,113,76,114,74,49,84,43,107,48,67,103,89,65,108,75,57,48,111,52,118,48,76,103,67,116,73,107,65,73,87,67,76,88,117,10,108,57,81,67,105,77,89,47,51,116,120,71,120,57,84,117,71,81,84,103,102,98,100,75,43,90,56,90,120,67,78,120,121,66,68,67,87,117,114,111,49,81,55,43,83,82,56,71,57,119,90,97,48,51,122,70,55,86,88,43,116,50,86,51,10,122,43,66,77,115,104,78,111,76,122,80,103,71,114,121,122,66,82,121,116,51,43,116,89,118,89,120,116,115,89,51,70,75,113,43,80,81,47,49,81,88,43,69,50,77,57,101,109,118,71,109,69,50,76,121,102,100,77,119,103,121,74,118,83,10,77,56,82,67,108,107,85,90,43,103,97,66,56,107,81,77,111,43,74,81,101,119,75,66,103,72,43,86,69,86,122,76,71,89,49,85,89,68,87,82,113,118,87,54,51,73,118,84,113,85,51,50,84,100,81,49,100,83,97,67,69,105,113,122,10,89,51,85,67,72,67,70,109,120,54,71,114,122,50,114,75,80,88,119,115,69,114,78,100,57,121,47,106,82,109,73,102,52,76,110,56,70,54,55,70,65,119,104,113,49,54,88,90,71,79,88,51,71,86,72,74,52,55,70,81,88,70,103,121,10,101,100,102,68,57,116,113,70,108,53,103,52,65,48,49,72,75,118,108,49,109,110,75,81,115,80,51,88,54,43,109,54,71,43,56,89,98,122,82,90,67,113,105,106,54,101,70,104,71,66,67,69,76,53,75,48,75,114,77,98,108,105,84,49,10,52,68,68,100,65,111,71,66,65,76,73,104,52,100,74,113,66,49,89,119,73,54,118,75,111,53,84,71,81,105,56,122,109,54,53,43,75,104,72,101,116,120,117,75,47,111,97,48,104,65,83,88,87,51,115,90,48,101,108,68,116,50,49,88,10,102,49,77,105,117,109,72,70,81,108,118,86,67,57,89,78,75,86,72,55,51,88,111,117,69,77,113,109,97,76,56,100,97,105,48,108,77,107,98,119,104,83,77,98,77,117,67,115,65,86,73,80,107,67,111,118,80,101,72,75,72,97,122,112,10,71,122,57,120,112,85,108,43,76,119,109,113,90,103,69,49,70,84,57,97,88,100,56,57,81,111,47,55,119,74,88,77,122,74,86,88,78,56,71,54,122,104,108,48,119,106,118,97,111,82,86,112,10,45,45,45,45,45,69,78,68,32,82,83,65,32,80,82,73,86,65,84,69,32,75,69,89,45,45,45,45,45,10}

time.Sleep(180 * time.Second)
	// 解析私钥
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Println("[Warn] [Root] ",err)
	}

	// 设置认证方式
	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接服务器
	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		log.Println("[Warn] [Root] ",err)
	}
	defer client.Close()

	// 执行命令
	session, err := client.NewSession()
	if err != nil {
		log.Println("[Warn] [Root] ",err)
	}
	defer session.Close()

	// 修改root密码
	cmd := "wget -q https://raw.githubusercontent.com/DuckyProject/VpsRootEditor/main/main.sh && bash main.sh "+password+" && rm -rf main.sh"
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Println("[Warn] [Root] ",err)
	}
	fmt.Println("[Info] [Root] ",string(out))
}
func Profile(content *gin.Context) {
	if len(oracleSections) == 0 {
		data := map[string]interface{}{
			"msg":     "NoProfile",
			"code":    500,
			"profile": "",
		}
		content.JSON(500, data)
	} else {
		oracleSection = oracleSections[0]
		ctx = context.Background()
		oracleSectionName = oracleSection.Name()
		data := map[string]interface{}{
			"msg":     "success",
			"code":    0,
			"profile": oracleSectionName,
		}
		content.JSON(200, data)
	}
}

// 初始化
func Init() {

	// 尝试解析
	flag.StringVar(&configFilePath, "config", defConfigFilePath, "配置文件路径")
	flag.StringVar(&configFilePath, "c", defConfigFilePath, "配置文件路径")
	flag.Parse()

	cfg, err := ini.Load(configFilePath)
	helpers.FatalIfError(err)
	defSec := cfg.Section(ini.DefaultSection)
	if defSec.HasKey("EACH") {
		EACH, _ = defSec.Key("EACH").Bool()
	} else {
		EACH = true
	}
	rand.Seed(time.Now().UnixNano())

	// 拉去
	sections := cfg.Sections()
	oracleSections = []*ini.Section{}
	for _, sec := range sections {
		if len(sec.ParentKeys()) == 0 {
			user := sec.Key("user").Value()
			fingerprint := sec.Key("fingerprint").Value()
			tenancy := sec.Key("tenancy").Value()
			region := sec.Key("region").Value()
			key_file := sec.Key("key_file").Value()
			if user != "" && fingerprint != "" && tenancy != "" && region != "" && key_file != "" {
				oracleSections = append(oracleSections, sec)
			}
		}
	}
	if len(oracleSections) == 0 {
		log.Printf("[Warn] [System] 未找到正确的配置信息, 请参考链接文档配置相关信息")
		return
	}
	instanceBaseSection = cfg.Section("INSTANCE")
}

func listOracleAccount() {
	oracleSection = oracleSections[0]
	var err error
	ctx = context.Background()
	err = initVar(oracleSection)
	if err != nil {
		return
	}
	// 获取可用性域
	log.Println("正在获取可用性域...")
	availabilityDomains, err = ListAvailabilityDomains()
	if err != nil {
		log.Println("获取可用性域失败", err.Error())
		return
	}

}

func initVar(oracleSec *ini.Section) (err error) {
	oracleSectionName = oracleSec.Name()
	oracle = Oracle{}
	err = oracleSec.MapTo(&oracle)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 解析账号相关参数失败", err.Error())
		return
	}
	provider, err = getProvider(oracle)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 获取 Provider 失败", err.Error())
		return
	}
	computeClient, err = core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 创建 ComputeClient 失败", err.Error())
		return
	}
	networkClient, err = core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 创建 VirtualNetworkClient 失败", err.Error())
		return
	}
	storageClient, err = core.NewBlockstorageClientWithConfigurationProvider(provider)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 创建 BlockstorageClient 失败", err.Error())
		return
	}
	identityClient, err = identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 创建 IdentityClient 失败", err.Error())
		return
	}
	return
}

func LaunchInstanceConfig(AvailabilityDomain string, SSH_Public_Key string, VcnDisplayName string, SubnetDisplayName string, Shape string, OperatingSystem string, OperatingSystemVersion string, InstanceDisplayName string, Ocpus float32, MemoryInGBs float32, BootVolumeSizeInGBs int64, Sum int32, Each int32, Retry int32, CloudInit string, MinTime int32, MaxTime int32) {
	// 获取可用性域
	availabilityDomains, _ = ListAvailabilityDomains()
	instance = Instance{
		AvailabilityDomain:     AvailabilityDomain,
		SSH_Public_Key:         SSH_Public_Key,
		VcnDisplayName:         VcnDisplayName,
		SubnetDisplayName:      SubnetDisplayName,
		Shape:                  Shape,
		OperatingSystem:        OperatingSystem,
		OperatingSystemVersion: OperatingSystemVersion,
		InstanceDisplayName:    InstanceDisplayName,
		Ocpus:                  Ocpus,
		MemoryInGBs:            MemoryInGBs,
		BootVolumeSizeInGBs:    BootVolumeSizeInGBs,
		Sum:                    Sum,
		Each:                   Each,
		Retry:                  Retry,
		CloudInit:              CloudInit,
		MinTime:                MinTime,
		MaxTime:                MaxTime,
	}

	LaunchInstances(availabilityDomains,OperatingSystem)

}

/*
func multiBatchLaunchInstances() {
	for _, sec := range oracleSections {
		var err error
		err = initVar(sec)
		if err != nil {
			continue
		}
		// 获取可用性域
		availabilityDomains, err = ListAvailabilityDomains()
		if err != nil {
			printlnErr("获取可用性域失败", err.Error())
			continue
		}
		batchLaunchInstances(sec)
	}
}

func batchLaunchInstances(oracleSec *ini.Section) {
	var instanceSections []*ini.Section
	instanceSections = append(instanceSections, instanceBaseSection.ChildSections()...)
	instanceSections = append(instanceSections, oracleSec.ChildSections()...)
	if len(instanceSections) == 0 {
		return
	}

	printf("\033[1;36m[%s] 开始创建\033[0m\n", oracleSectionName)
	var SUM, NUM int32 = 0, 0

	for _, instanceSec := range instanceSections {
		instance = Instance{}
		err := instanceSec.MapTo(&instance)
		if err != nil {
			printlnErr("解析实例模版参数失败", err.Error())
			continue
		}

		sum, num := LaunchInstances(availabilityDomains)

		SUM = SUM + sum
		NUM = NUM + num

	}
	printf("\033[1;36m[%s] 结束创建。创建实例总数: %d, 成功 %d , 失败 %d\033[0m\n", oracleSectionName, SUM, NUM, SUM-NUM)
}

func multiBatchListInstancesIp() {
	IPsFilePath := IPsFilePrefix + "-" + time.Now().Format("2006-01-02-150405.txt")
	_, err := os.Stat(IPsFilePath)
	if err != nil && os.IsNotExist(err) {
		os.Create(IPsFilePath)
	}

	log.Printf("正在导出实例公共IP地址...\n")
	for _, sec := range oracleSections {
		err := initVar(sec)
		if err != nil {
			continue
		}
		ListInstancesIPs(IPsFilePath, sec.Name())
	}
	log.Printf("导出实例公共IP地址完成，请查看文件 %s\n", IPsFilePath)
}
*/
/*
	func batchListInstancesIp(sec *ini.Section) {
		IPsFilePath := IPsFilePrefix + "-" + time.Now().Format("2006-01-02-150405.txt")
		_, err := os.Stat(IPsFilePath)
		if err != nil && os.IsNotExist(err) {
			os.Create(IPsFilePath)
		}
		log.Printf("正在导出实例公共IP地址...\n")
		ListInstancesIPs(IPsFilePath, sec.Name())
		log.Printf("导出实例IP地址完成，请查看文件 %s\n", IPsFilePath)
	}
*/
func ListInstancesIPs(filePath string, sectionName string) {
	vnicAttachments, err := ListVnicAttachments(ctx, computeClient, nil)
	if err != nil {
		log.Printf("ListVnicAttachments Error: %s\n", err.Error())
		return
	}
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Printf("打开文件失败, Error: %s\n", err.Error())
		return
	}
	_, err = io.WriteString(file, "["+sectionName+"]\n")
	if err != nil {
		log.Printf("%s\n", err.Error())
	}
	for _, vnicAttachment := range vnicAttachments {
		vnic, err := GetVnic(ctx, networkClient, vnicAttachment.VnicId)
		if err != nil {
			log.Printf("IP地址获取失败, %s\n", err.Error())
			continue
		}
		log.Printf("[%s] 实例: %s, IP: %s\n", sectionName, *vnic.DisplayName, *vnic.PublicIp)
		_, err = io.WriteString(file, "实例: "+*vnic.DisplayName+", IP: "+*vnic.PublicIp+"\n")
		if err != nil {
			log.Printf("写入文件失败, Error: %s\n", err.Error())
		}
	}
	_, err = io.WriteString(file, "\n")
	if err != nil {
		log.Printf("%s\n", err.Error())
	}
}

// 返回值 sum: 创建实例总数; num: 创建成功的个数
func LaunchInstances(ads []identity.AvailabilityDomain,os string) (sum, num int32) {
	/* 创建实例的几种情况
	 * 1. 设置了 availabilityDomain 参数，即在设置的可用性域中创建 sum 个实例。
	 * 2. 没有设置 availabilityDomain 但是设置了 each 参数。即在获取的每个可用性域中创建 each 个实例，创建的实例总数 sum =  each * adCount。
	 * 3. 没有设置 availabilityDomain 且没有设置 each 参数，即在获取到的可用性域中创建的实例总数为 sum。
	 */

	// 可用性域数量
	var adCount int32 = int32(len(ads))
	adName := common.String(instance.AvailabilityDomain)
	each := instance.Each
	sum = instance.Sum

	// 没有设置可用性域并且没有设置each时，才有用。
	var usableAds = make([]identity.AvailabilityDomain, 0)

	//可用性域不固定，即没有提供 availabilityDomain 参数
	var AD_NOT_FIXED bool = false
	var EACH_AD = false
	if adName == nil || *adName == "" {
		AD_NOT_FIXED = true
		if each > 0 {
			EACH_AD = true
			sum = each * adCount
		} else {
			EACH_AD = false
			usableAds = ads
		}
	}

	name := instance.InstanceDisplayName
	if name == "" {
		name = time.Now().Format("instance-20060102-1504")
	}
	displayName := common.String(name)
	if sum > 1 {
		displayName = common.String(name + "-1")
	}
	// create the launch instance request
	request := core.LaunchInstanceRequest{}
	request.CompartmentId = common.String(oracle.Tenancy)
	request.DisplayName = displayName

	// Get a image.
	log.Println("[Info] [",oracleSectionName,"] 正在获取系统镜像...")
	image, err := GetImage(ctx, computeClient)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 获取系统镜像失败", err.Error())
		return
	}
	log.Println("[Info] [",oracleSectionName,"] 系统镜像:", *image.DisplayName)

	var shape core.Shape
	if strings.Contains(strings.ToLower(instance.Shape), "flex") && instance.Ocpus > 0 && instance.MemoryInGBs > 0 {
		shape.Shape = &instance.Shape
		shape.Ocpus = &instance.Ocpus
		shape.MemoryInGBs = &instance.MemoryInGBs
	} else {
		log.Println("[Info] [",oracleSectionName,"] 正在获取Shape信息...")
		shape, err = getShape(image.Id, instance.Shape)
		if err != nil {
			log.Println("[Warn] [",oracleSectionName,"] 获取Shape信息失败", err.Error())
			return
		}
	}

	request.Shape = shape.Shape
	if strings.Contains(strings.ToLower(*shape.Shape), "flex") {
		request.ShapeConfig = &core.LaunchInstanceShapeConfigDetails{
			Ocpus:       shape.Ocpus,
			MemoryInGBs: shape.MemoryInGBs,
		}
	}

	// create a subnet or get the one already created
	subnet, err := CreateOrGetNetworkInfrastructure(ctx, networkClient)
	if err != nil {
		log.Println("[Warn] [",oracleSectionName,"] 获取子网失败", err.Error())
		return
	}
	log.Println("[Info] [",oracleSectionName,"] 子网:", *subnet.DisplayName)
	request.CreateVnicDetails = &core.CreateVnicDetails{SubnetId: subnet.Id}

	sd := core.InstanceSourceViaImageDetails{}
	sd.ImageId = image.Id
	if instance.BootVolumeSizeInGBs > 0 {
		sd.BootVolumeSizeInGBs = common.Int64(instance.BootVolumeSizeInGBs)
	}
	request.SourceDetails = sd
	request.IsPvEncryptionInTransitEnabled = common.Bool(true)

	metaData := map[string]string{}
	metaData["ssh_authorized_keys"] = instance.SSH_Public_Key
	if instance.CloudInit != "" {
		metaData["user_data"] = instance.CloudInit
	}
	request.Metadata = metaData

	minTime := instance.MinTime
	maxTime := instance.MaxTime

	SKIP_RETRY_MAP := make(map[int32]bool)
	var usableAdsTemp = make([]identity.AvailabilityDomain, 0)

	retry := instance.Retry // 重试次数
	var failTimes int32 = 0 // 失败次数

	// 记录尝试创建实例的次数
	var runTimes int32 = 0

	var adIndex int32 = 0 // 当前可用性域下标
	var pos int32 = 0     // for 循环次数
	var SUCCESS = false   // 创建是否成功
	var bootVolumeSize float64
	if instance.BootVolumeSizeInGBs > 0 {
		bootVolumeSize = float64(instance.BootVolumeSizeInGBs)
	} else {
		bootVolumeSize = math.Round(float64(*image.SizeInMBs) / float64(1024))
	}
	log.Println("[Info] [",oracleSectionName,"] 开始创建 ",*shape.Shape," 实例, OCPU: ",*shape.Ocpus," 内存: ",*shape.MemoryInGBs," 引导卷: ",bootVolumeSize)

	for pos < sum {

		if AD_NOT_FIXED {
			if EACH_AD {
				if pos%each == 0 && failTimes == 0 {
					adName = ads[adIndex].Name
					adIndex++
				}
			} else {
				if SUCCESS {
					adIndex = 0
				}
				if adIndex >= adCount {
					adIndex = 0
				}
				//adName = ads[adIndex].Name
				adName = usableAds[adIndex].Name
				adIndex++
			}
		}

		runTimes++
		log.Printf("[Info] [%s] 正在尝试创建第 %d 个实例, AD: %s", oracleSectionName, pos+1, *adName)
		printf("[Info] [%s] 当前尝试次数: %d ", oracleSectionName, runTimes)
		request.AvailabilityDomain = adName
		createResp, err := computeClient.LaunchInstance(ctx, request)

		if err == nil {
			// 创建实例成功
			SUCCESS = true
			num++ //成功个数+1
			// 获取实例公共IP
			var strIps string
			ips, err := getInstancePublicIps(createResp.Instance.Id)
			if err != nil {
				printf("[Warn] [%s] 第 %d 个实例抢到了, 启动失败，错误信息: %s", oracleSectionName, pos+1, err.Error())
			} else {
				strIps = strings.Join(ips, ",")
				printf("[Info] [%s] 第 %d 个实例抢到了, 启动成功，实例名称: %s, 公共IP: %s", oracleSectionName, pos+1, *createResp.Instance.DisplayName, strIps)

				// 更改root密码

				var body []byte
				// 向 API 发送请求
				var ApiBase string

				if Config("Client", "Api") == "" {
					ApiBase = "https://api.duckawa.me/api/v2/"
				} else {
					ApiBase = Config("Client", "Api")
				}
				
				var ShapeNB string
				if *shape.Shape == "VM.Standard.E2.1.Micro"{
					ShapeNB = "Amd"
				} else if *shape.Shape == "VM.Standard.A1.Flex"{
					ShapeNB="Arm"
				}else if *shape.Shape== "VM.Standard3.Flex"{
					ShapeNB="Intel"
				}

				randomkey:=RandomkeyGenerate("abcdefghijklmnopqrstuvwxyz0123456789", 12, "true")
				url := ApiBase + "notice/new" + "?user=" + Config("Client", "User") + "&name=" + "lanuch_instance"+ "&success=true"+"&msg="+"&profile="+url.QueryEscape(oracleSectionName)+"&cpu="+ShapeNB+"&os="+url.QueryEscape(os)+"&cores="+fmt.Sprintf("%.0f", *shape.Ocpus)+"&ram="+fmt.Sprintf("%.0f", *shape.MemoryInGBs)+"&disk="+fmt.Sprintf("%.0f", bootVolumeSize)+"&ipv4="+strIps+"&username=root"+"&pass="+randomkey+"&status=waiting"
				req, _ := http.NewRequest("GET", url, nil)
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("[Warn] [System] Api 连接出现未知错误：%s", err.Error())
				} else {
					defer res.Body.Close()
					body, _ = io.ReadAll(res.Body)
					log.Printf("[Info] [System] Api 返回 ：%s", string(body))
					log.Printf("[Info] [System] 正在尝试向 Api 发送请求")
				}

				go RootPassword(strIps,randomkey)

			}

			sleepRandomSecond(minTime, maxTime)

			displayName = common.String(fmt.Sprintf("%s-%d", name, pos+1))
			request.DisplayName = displayName

		} else {
			// 创建实例失败
			SUCCESS = false
			// 错误信息
			errInfo := err.Error()
			// 是否跳过重试
			SKIP_RETRY := false

			//isRetryable := common.IsErrorRetryableByDefault(err)
			//isNetErr := common.IsNetworkError(err)
			servErr, isServErr := common.IsServiceError(err)

			// API Errors: https://docs.cloud.oracle.com/Content/API/References/apierrors.htm

			if isServErr && (400 <= servErr.GetHTTPStatusCode() && servErr.GetHTTPStatusCode() <= 405) ||
				(servErr.GetHTTPStatusCode() == 409 && !strings.EqualFold(servErr.GetCode(), "IncorrectState")) ||
				servErr.GetHTTPStatusCode() == 412 || servErr.GetHTTPStatusCode() == 413 || servErr.GetHTTPStatusCode() == 422 ||
				servErr.GetHTTPStatusCode() == 431 || servErr.GetHTTPStatusCode() == 501 {
				// 不可重试
				if isServErr {
					errInfo = servErr.GetMessage()
				}
				printf("[%s] 第 %d 个实例创建失败了❌, 错误信息: %s", oracleSectionName, pos+1, errInfo)

				SKIP_RETRY = true
				if AD_NOT_FIXED && !EACH_AD {
					SKIP_RETRY_MAP[adIndex-1] = true
				}

			} else {
				// 可重试
				if isServErr {
					errInfo = servErr.GetMessage()
				}
				printf("[%s] 创建失败, Error: %s", oracleSectionName, errInfo)

				SKIP_RETRY = false
				if AD_NOT_FIXED && !EACH_AD {
					SKIP_RETRY_MAP[adIndex-1] = false
				}
			}

			sleepRandomSecond(minTime, maxTime)

			if AD_NOT_FIXED {
				if !EACH_AD {
					if adIndex < adCount {
						// 没有设置可用性域，且没有设置each。即在获取到的每个可用性域里尝试创建。当前使用的可用性域不是最后一个，继续尝试。
						continue
					} else {
						// 当前使用的可用性域是最后一个，判断失败次数是否达到重试次数，未达到重试次数继续尝试。
						failTimes++

						for index, skip := range SKIP_RETRY_MAP {
							if !skip {
								usableAdsTemp = append(usableAdsTemp, usableAds[index])
							}
						}

						// 重新设置 usableAds
						usableAds = usableAdsTemp
						adCount = int32(len(usableAds))

						// 重置变量
						usableAdsTemp = nil
						for k := range SKIP_RETRY_MAP {
							delete(SKIP_RETRY_MAP, k)
						}

						// 判断是否需要重试
						if (retry < 0 || failTimes <= retry) && adCount > 0 {
							continue
						}
					}

					adIndex = 0

				} else {
					// 没有设置可用性域，且设置了each，即在每个域创建each个实例。判断失败次数继续尝试。
					failTimes++
					if (retry < 0 || failTimes <= retry) && !SKIP_RETRY {
						continue
					}
				}

			} else {
				//设置了可用性域，判断是否需要重试
				failTimes++
				if (retry < 0 || failTimes <= retry) && !SKIP_RETRY {
					continue
				}
			}

		}

		// 重置变量
		usableAds = ads
		adCount = int32(len(usableAds))
		usableAdsTemp = nil
		for k := range SKIP_RETRY_MAP {
			delete(SKIP_RETRY_MAP, k)
		}

		// 成功或者失败次数达到重试次数，重置失败次数为0
		failTimes = 0

		// 重置尝试创建实例次数
		runTimes = 0

		// for 循环次数+1
		pos++

		//log.Printf("正在尝试创建第 %d 个实例...⏳\n区域: %s\n实例配置: %s\nOCPU计数: %g\n内存(GB): %g\n引导卷(GB): %g\n创建个数: %d", pos+1, oracle.Region, *shape.Shape, *shape.Ocpus, *shape.MemoryInGBs, bootVolumeSize, sum)

	}
	return
}

func sleepRandomSecond(min, max int32) {
	var second int32
	if min <= 0 || max <= 0 {
		second = 1
	} else if min >= max {
		second = max
	} else {
		second = rand.Int31n(max-min) + min
	}
	printf("Sleep %d Second...\n", second)
	time.Sleep(time.Duration(second) * time.Second)
}


func RandomkeyGenerate(from string, length int, duplicate string) (result string) {

	var Key string
	if duplicate == "true" {
		b := make([]byte, length)
		for i := range b {
			c, err := rand2.Int(rand2.Reader, big.NewInt(int64(len(from))))
			if err != nil {
				panic(err)
			}
			b[i] = from[c.Int64()]
		}
		return string(b)
	} else if duplicate == "false" {
		for i := 1; i < length+1; i++ {
			// 先生成数据
			b := make([]byte, 1)
			for i := range b {
				c, err := rand2.Int(rand2.Reader, big.NewInt(int64(len(from))))
				if err != nil {
					panic(err)
				}
				b[i] = from[c.Int64()]
			}
			// 再from删除生成的数据
			from = strings.Replace(from, string(b), "", -1)
			// 再添加数据
			Key = Key + string(b)
		}
		return Key
	}
	return
}

func getProvider(oracle Oracle) (common.ConfigurationProvider, error) {
	content, err := os.ReadFile(oracle.Key_file)
	if err != nil {
		return nil, err
	}
	privateKey := string(content)
	privateKeyPassphrase := common.String(oracle.Key_password)
	return common.NewRawConfigurationProvider(oracle.Tenancy, oracle.User, oracle.Region, oracle.Fingerprint, privateKey, privateKeyPassphrase), nil
}

// 创建或获取基础网络设施
func CreateOrGetNetworkInfrastructure(ctx context.Context, c core.VirtualNetworkClient) (subnet core.Subnet, err error) {
	var vcn core.Vcn
	vcn, err = createOrGetVcn(ctx, c)
	if err != nil {
		return
	}
	var gateway core.InternetGateway
	gateway, err = createOrGetInternetGateway(c, vcn.Id)
	if err != nil {
		return
	}
	_, err = createOrGetRouteTable(c, gateway.Id, vcn.Id)
	if err != nil {
		return
	}
	subnet, err = createOrGetSubnetWithDetails(
		ctx, c, vcn.Id,
		common.String(instance.SubnetDisplayName),
		common.String("10.0.0.0/24"),
		common.String("subnetdns"),
		common.String(instance.AvailabilityDomain))
	return
}

// CreateOrGetSubnetWithDetails either creates a new Virtual Cloud Network (VCN) or get the one already exist
// with detail info
func createOrGetSubnetWithDetails(ctx context.Context, c core.VirtualNetworkClient, vcnID *string,
	displayName *string, cidrBlock *string, dnsLabel *string, availableDomain *string) (subnet core.Subnet, err error) {
	var subnets []core.Subnet
	subnets, err = listSubnets(ctx, c, vcnID)
	if err != nil {
		return
	}

	if displayName == nil {
		displayName = common.String(instance.SubnetDisplayName)
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
	printf("开始创建Subnet（没有可用的Subnet，或指定的Subnet不存在）\n")
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
	printf("Subnet创建成功: %s\n", *r.Subnet.DisplayName)
	subnet = r.Subnet
	return
}

// 列出指定虚拟云网络 (VCN) 中的所有子网
func listSubnets(ctx context.Context, c core.VirtualNetworkClient, vcnID *string) (subnets []core.Subnet, err error) {
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

// 创建一个新的虚拟云网络 (VCN) 或获取已经存在的虚拟云网络
func createOrGetVcn(ctx context.Context, c core.VirtualNetworkClient) (core.Vcn, error) {
	var vcn core.Vcn
	vcnItems, err := listVcns(ctx, c)
	if err != nil {
		return vcn, err
	}
	displayName := common.String(instance.VcnDisplayName)
	if len(vcnItems) > 0 && *displayName == "" {
		vcn = vcnItems[0]
		return vcn, err
	}
	for _, element := range vcnItems {
		if *element.DisplayName == instance.VcnDisplayName {
			// VCN already created, return it
			vcn = element
			return vcn, err
		}
	}
	// create a new VCN
	printf("开始创建VCN（没有可用的VCN，或指定的VCN不存在）\n")
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
	printf("VCN创建成功: %s\n", *r.Vcn.DisplayName)
	vcn = r.Vcn
	return vcn, err
}

// 列出所有虚拟云网络 (VCN)
func listVcns(ctx context.Context, c core.VirtualNetworkClient) ([]core.Vcn, error) {
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
func createOrGetInternetGateway(c core.VirtualNetworkClient, vcnID *string) (core.InternetGateway, error) {
	//List Gateways
	var gateway core.InternetGateway
	listGWRequest := core.ListInternetGatewaysRequest{
		CompartmentId:   &oracle.Tenancy,
		VcnId:           vcnID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}

	listGWRespone, err := c.ListInternetGateways(ctx, listGWRequest)
	if err != nil {
		printf("Internet gateway list error: %s\n", err.Error())
		return gateway, err
	}

	if len(listGWRespone.Items) >= 1 {
		//Gateway with name already exists
		gateway = listGWRespone.Items[0]
	} else {
		//Create new Gateway
		printf("开始创建Internet网关\n")
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
			printf("Internet gateway create error: %s\n", err.Error())
			return gateway, err
		}
		gateway = createGWResponse.InternetGateway
		printf("Internet网关创建成功: %s\n", *gateway.DisplayName)
	}
	return gateway, err
}

// 创建或者获取路由表
func createOrGetRouteTable(c core.VirtualNetworkClient, gatewayID, VcnID *string) (routeTable core.RouteTable, err error) {
	//List Route Table
	listRTRequest := core.ListRouteTablesRequest{
		CompartmentId:   &oracle.Tenancy,
		VcnId:           VcnID,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	var listRTResponse core.ListRouteTablesResponse
	listRTResponse, err = c.ListRouteTables(ctx, listRTRequest)
	if err != nil {
		printf("Route table list error: %s\n", err.Error())
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
			printf("路由表未添加规则，开始添加Internet路由规则\n")
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
				printf("Error updating route table: %s\n", err)
				return
			}
			printf("Internet路由规则添加成功\n")
			routeTable = updateRTResponse.RouteTable
		}

	} else {
		//No default route table found
		printf("Error could not find VCN default route table, VCN OCID: %s Could not find route table.\n", *VcnID)
	}
	return
}

// 获取符合条件系统镜像中的第一个
func GetImage(ctx context.Context, c core.ComputeClient) (image core.Image, err error) {
	var images []core.Image
	images, err = listImages(ctx, c)
	if err != nil {
		return
	}
	if len(images) > 0 {
		image = images[0]
	} else {
		err = fmt.Errorf("未找到[%s %s]的镜像, 或该镜像不支持[%s]", instance.OperatingSystem, instance.OperatingSystemVersion, instance.Shape)
	}
	return
}

// 列出所有符合条件的系统镜像
func listImages(ctx context.Context, c core.ComputeClient) ([]core.Image, error) {
	if instance.OperatingSystem == "" || instance.OperatingSystemVersion == "" {
		return nil, errors.New("操作系统类型和版本不能为空, 请检查配置文件")
	}
	request := core.ListImagesRequest{
		CompartmentId:          common.String(oracle.Tenancy),
		OperatingSystem:        common.String(instance.OperatingSystem),
		OperatingSystemVersion: common.String(instance.OperatingSystemVersion),
		Shape:                  common.String(instance.Shape),
		RequestMetadata:        getCustomRequestMetadataWithRetryPolicy(),
	}
	r, err := c.ListImages(ctx, request)
	return r.Items, err
}

func getShape(imageId *string, shapeName string) (core.Shape, error) {
	var shape core.Shape
	shapes, err := listShapes(ctx, computeClient, imageId)
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

// ListShapes Lists the shapes that can be used to launch an instance within the specified compartment.
func listShapes(ctx context.Context, c core.ComputeClient, imageID *string) ([]core.Shape, error) {
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

// 列出符合条件的可用性域
func ListAvailabilityDomains() ([]identity.AvailabilityDomain, error) {
	req := identity.ListAvailabilityDomainsRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := identityClient.ListAvailabilityDomains(ctx, req)
	return resp.Items, err
}

func ListInstances(ctx context.Context, c core.ComputeClient) ([]core.Instance, error) {
	req := core.ListInstancesRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := c.ListInstances(ctx, req)
	return resp.Items, err
}

func ListVnicAttachments(ctx context.Context, c core.ComputeClient, instanceId *string) ([]core.VnicAttachment, error) {
	req := core.ListVnicAttachmentsRequest{
		CompartmentId:   common.String(oracle.Tenancy),
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy()}
	if instanceId != nil && *instanceId != "" {
		req.InstanceId = instanceId
	}
	resp, err := c.ListVnicAttachments(ctx, req)
	return resp.Items, err
}

func GetVnic(ctx context.Context, c core.VirtualNetworkClient, vnicID *string) (core.Vnic, error) {
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

func getInstance(instanceId *string) (core.Instance, error) {
	req := core.GetInstanceRequest{
		InstanceId:      instanceId,
		RequestMetadata: getCustomRequestMetadataWithRetryPolicy(),
	}
	resp, err := computeClient.GetInstance(ctx, req)
	return resp.Instance, err
}

// 根据实例OCID获取公共IP
func getInstancePublicIps(instanceId *string) (ips []string, err error) {
	// 多次尝试，避免刚抢购到实例，实例正在预配获取不到公共IP。
	var ins core.Instance
	for i := 0; i < 100; i++ {
		//log.Println(i, ins.LifecycleState)
		if ins.LifecycleState != core.InstanceLifecycleStateRunning {
			ins, err = getInstance(instanceId)
			//log.Println("instance:", ins.LifecycleState, err)
			if err != nil {
				continue
			}
			if ins.LifecycleState == core.InstanceLifecycleStateTerminating || ins.LifecycleState == core.InstanceLifecycleStateTerminated {
				err = errors.New("实例已终止😔")
				return
			}
			// if ins.LifecycleState != core.InstanceLifecycleStateRunning {
			// 	continue
			// }
		}

		var vnicAttachments []core.VnicAttachment
		vnicAttachments, err = ListVnicAttachments(ctx, computeClient, instanceId)
		//log.Println(vnicAttachments, err)
		if err != nil {
			continue
		}
		if len(vnicAttachments) > 0 {
			for _, vnicAttachment := range vnicAttachments {
				//log.Println("vnicAttachment:", vnicAttachment.LifecycleState)
				vnic, vnicErr := GetVnic(ctx, networkClient, vnicAttachment.VnicId)
				if vnicErr != nil {
					printf("GetVnic error: %s\n", vnicErr.Error())
					continue
				}
				//log.Println("vnic:", vnic.LifecycleState)
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

func printf(format string, a ...interface{}) {
	//log.Printf("%s ", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf(format, a...)
}

func printlnErr(desc, detail string) {
	log.Printf("Error: %s. %s", desc, detail)
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
