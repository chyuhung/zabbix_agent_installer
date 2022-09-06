package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"time"

	"github.com/go-ini/ini"
)

// zabbix agent user
// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
var (
	Cloud = "cloud"
)

type agentConfig struct {
	serverIP   string
	serverPort string
	agentUser  string
	agentDir   string
}

func NewAgent() *agentConfig {
	return &agentConfig{}
}
func (a *agentConfig) SetServerIP(ip string) {
	a.serverIP = ip
}
func (a *agentConfig) SetServerPort(port string) {
	a.serverPort = port
}
func (a *agentConfig) SetAgentUser(user string) {
	a.agentUser = user
}
func (a *agentConfig) SetAgentDir(dir string) {
	a.agentDir = dir
}

// 配置安装用户信息
func (a *agentConfig) setParams(serverIP string, port string, user string, directory string) {
	a.SetServerIP(serverIP)
	a.SetServerPort(port)
	a.SetAgentDir(directory)
}
func (a *agentConfig) GetServerIP() string {
	return a.serverIP
}
func (a *agentConfig) GetServerPort() string {
	return a.serverPort
}
func (a *agentConfig) GetAgentUser() string {
	return a.agentUser
}
func (a *agentConfig) GetAgentDir() string {
	return a.agentDir
}

// 帮助信息
func helpInfo() {
}

// 日志打印
func logger(level string, log string) {
	if level != "" {
		fmt.Printf("[%s] %s\n", level, log)
	} else {
		fmt.Printf("[%s] %s\n", "SYSTEM", log)
	}
}

// 获取 ip
func getClientIP() string {
	var ip string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger("", err.Error())
		os.Exit(1)
	}
	for _, addr := range addrs {
		// 判断是否回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	if ip == "" {
		logger("ERROR", "获取ip失败")
		os.Exit(1)
	}
	return ip
}

// 获取当前用户用户名
func getUser() string {
	currentUser, err := user.Current()
	if err != nil {
		logger("", err.Error())
	}
	return currentUser.Name
}

// 获取当前用户家目录
func getUserDir() string {
	var dir string
	currentUser, err := user.Current()
	if err != nil {
		logger("", err.Error())
	}
	if d := currentUser.HomeDir; d != "" {
		dir = d
	} else {
		logger("ERROR", "获取homeDir失败")
		os.Exit(1)
	}
	return dir
}

// 判断ip是否合规
func isIPv4(ipv4 string) bool {
	ip := net.ParseIP(ipv4)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	return ip != nil
}

// 测试ip是否可达
func isIPReachable(ipv4 string, port string) bool {
	addr := net.JoinHostPort(ipv4, port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		logger("", err.Error())
		return false
	}
	defer conn.Close()
	return conn != nil
}

// 组装配置必要参数
func getParams() (serverIP string, port string, username string, directory string) {
	// 接受命令
	flag.StringVar(&serverIP, "s", "", "zabbix server ip,you must input server ip.")
	flag.StringVar(&port, "p", "8001", "zabbix server port.")
	flag.StringVar(&username, "u", "cloud", "zabbix agent username.")
	flag.StringVar(&directory, "d", "", "zabbix agent directory,default is current user's home directory.")
	// 转换
	flag.Parse()

	// 补充空值参数
	// server ip,必要参数，不指定则panic
	if serverIP == "" {
		logger("ERROR", "缺少必要参数serverip")
		os.Exit(1)
	}
	if !isIPv4(serverIP) {
		logger("ERROR", "非法ip")
		os.Exit(1)
	}
	// port
	if isIPReachable(serverIP, port) {
		logger("INFO", fmt.Sprintf("%s:%s is reachable", serverIP, port))
	} else {
		logger("INFO", fmt.Sprintf("%s:%s is unreachable", serverIP, port))

	}

	// username
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	if username != getUser() && username == Cloud {
		logger("ERROR", "请切换到默认用户再安装")
		os.Exit(1)
	}
	if username != getUser() && username != Cloud {
		logger("ERROR", "请切换到指定用户再安装")
		os.Exit(1)
	}
	// directory
	if directory == "" {
		directory = getUserDir()
	}
	return serverIP, port, username, directory
}

// 下载安装包
func downloadPackage(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 创建文件，从url中读取文件名
	filename := path.Base(url)
	out, err := os.OpenFile(savePath+filename, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// 解压安装包
func unzipPackage(filename string, directory string) error {
	// 获取运行路径,构造文件绝对路径
	cd, err := os.Getwd()
	if err != nil {
		return err
	}
	filePath := cd + "/" + filename
	// 检查文件是否存在
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		return err
	}
	// 使用linux命令处理
	cmd := exec.Command("tar", "--directory", directory, "-xzf", filename)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func setINI(cfg *ini.File) {
	// 	$ cat etc/zabbix_agentd.conf|grep -vE "#|^$"
	// PidFile=%change_basepath%/zabbix_agentd.pid
	// LogFile=%change_basepath%/zabbix_agentd.log
	// StartAgents=0
	// ServerActive=%change_serverip%
	// Hostname=%change_hostname%
	// HostMetadataItem=system.uname
	// BufferSize=200
	// Include=%change_basepath%/etc/zabbix_agentd.conf.d/
	// UnsafeUserParameters=1

	cfg.Section("").Key("ServerActive").SetValue(NewAgent().serverIP)
}

func loadINI(filename string) error {
	cd, err := os.Getwd()
	if err != nil {
		return err
	}
	filePath := cd + "/zabbix_agentd/" + filename
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		return err
	}
	cfg, err := ini.Load(filePath)
	if err != nil {
		return err
	}
	setINI(cfg)
	return nil

}

// 获取操作系统类型
func getOSType() string {
	return ""
}

// 设置安装目录
func setDir() {}
func main() {
	// 创建实例存储关键参数
	agent := NewAgent()
	// 获取关键参数
	serverIP, port, username, directory := getParams()
	// 设置实例关键参数
	agent.setParams(serverIP, port, username, directory)
	// 输出配置信息
	fmt.Println(agent.serverIP)
	fmt.Println(agent.agentUser)
	fmt.Println(agent.agentDir)
	filename := "zabbix-agentd-5.0.14-1.linux.x86_64.tar.gz"
	err := unzipPackage(filename, ".")
	if err != nil {
		logger("", err.Error())
	}
}
