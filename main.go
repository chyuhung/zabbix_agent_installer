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
	"strings"
	"time"

	"github.com/go-ini/ini"
)

// zabbix agent user
// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
var (
	Cloud       = "cloud"
	LinuxURL    = "http://10.191.22.9:8001/software/zabbix-agent4.0/zabbix_agentd_linux/"
	WinURL      = "http://10.191.22.9:8001/software/zabbix-agent4.0/zabbix_agentd_windows/"
	PackageName = "zabbix-agentd-5.0.14-1.linux.x86_64.tar.gz"
)
var (
	ServerIP   string
	ServerPort string
	AgentUser  string
	AgentDir   string
	AgentIP    string
)

// 日志打印
func logger(level string, log string) {
	if level != "" {
		fmt.Printf("[%s] %s\n", level, log)
	} else {
		fmt.Printf("[%s] %s\n", "SYSTEM", log)
	}
}

// 获取主ip
func getAgentIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		logger("", err.Error())
		os.Exit(1)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ipaddr := strings.Split(localAddr.String(), ":")[0]
	return ipaddr
}

// 获取当前用户用户名
func getCurrUser() string {
	currentUser, err := user.Current()
	if err != nil {
		logger("", err.Error())
	}
	return currentUser.Name
}

// 获取当前用户家目录
func getCurrUserHomeDir() string {
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
func isReachable(ipv4 string, port string) bool {
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
func scanParams() (serverIP string, serverPort string, agentUser string, agentDir string, agentIP string) {
	// 接受命令
	flag.StringVar(&serverIP, "s", "", "zabbix server ip,you must input server ip.")
	flag.StringVar(&serverPort, "p", "8001", "zabbix server port.")
	flag.StringVar(&agentUser, "u", "cloud", "zabbix agent user.")
	flag.StringVar(&agentDir, "d", "", "zabbix agent directory,default is current user's home directory.")
	flag.StringVar(&agentIP, "i", "", "zabbix agent ip.")
	// 转换
	flag.Parse()
	// 补充空值参数
	// serverIP,必要参数，不指定则panic
	if serverIP == "" {
		logger("ERROR", "缺少必要参数serverip")
		os.Exit(1)
	}
	if !isIPv4(serverIP) {
		logger("ERROR", "非法ip")
		os.Exit(1)
	}
	// serverPort
	if isReachable(serverIP, serverPort) {
		logger("INFO", fmt.Sprintf("%s:%s is reachable", serverIP, serverPort))
	} else {
		logger("INFO", fmt.Sprintf("%s:%s is unreachable", serverIP, serverPort))
	}
	// agentUser
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	if agentUser != getCurrUser() && agentUser == Cloud {
		logger("ERROR", "请切换到默认用户再安装")
		os.Exit(1)
	}
	if agentUser != getCurrUser() && agentUser != Cloud {
		logger("ERROR", "请切换到指定用户再安装")
		os.Exit(1)
	}
	// agentDir
	if agentDir == "" {
		agentDir = getCurrUserHomeDir()
	}
	// agentIP
	agentIP = getAgentIP()
	return serverIP, serverPort, agentUser, agentDir, agentIP
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
		err := downloadPackage(LinuxURL, filePath)
		if err != nil {
			logger("", err.Error())
			os.Exit(1)
		}
	}
	// 使用linux命令处理
	cmd := exec.Command("tar", "--directory", directory, "-xzf", filename)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func setINI(filePath string) error {
	cd, err := os.Getwd()
	if err != nil {
		return err
	}
	fileFullPath := cd + filePath
	_, err = os.Stat(fileFullPath)
	if os.IsNotExist(err) {
		return err
	}
	cfg, err := ini.Load(fileFullPath)
	if err != nil {
		return err
	}
	cfg.Section("").Key("ServerActive").SetValue(ServerIP)
	cfg.Section("").Key("Hostname").SetValue(AgentIP)
	cfg.Section("").Key("Include").SetValue(AgentDir + "/etc/zabbix_agentd.conf.d/")
	cfg.Section("").Key("PidFile").SetValue(AgentDir + "/zabbix_agentd.pid")
	cfg.Section("").Key("LogFile").SetValue(AgentDir + "/zabbix_agentd.log")
	err = cfg.SaveTo(fileFullPath)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// 获取关键参数
	ServerIP, ServerPort, AgentUser, AgentDir, AgentIP := scanParams()
	// 输出配置信息
	logger("INFO", fmt.Sprintf("ServerIP:%s", ServerIP))
	logger("INFO", fmt.Sprintf("ServerPort:%s", ServerPort))
	logger("INFO", fmt.Sprintf("AgentUser:%s", AgentUser))
	logger("INFO", fmt.Sprintf("AgentDir:%s", AgentDir))
	logger("INFO", fmt.Sprintf("AgentIP:%s", AgentIP))
	// 解压安装包
	err := unzipPackage(PackageName, ".")
	if err != nil {
		logger("", err.Error())
	}
	setINI("/zabbix_agentd/etc/abbix_agentd.conf")
	fmt.Println("done")
}
