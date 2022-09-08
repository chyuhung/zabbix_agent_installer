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
	"path/filepath"
	"strings"
	"time"
	"zabbix_agent_installer/utils"

	"github.com/go-ini/ini"
)

// zabbix agent user
// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
var (
	DefaultUser = "cloud"
	LinuxURL    = "http://10.191.22.9:8001/software/zabbix-agent4.0/zabbix_agentd_linux/"
	WinURL      = "http://10.191.22.9:8001/software/zabbix-agent4.0/zabbix_agentd_windows/"
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
	if level == "" {
		level = "SYSTEM"
	}
	fmt.Printf("[%s] %s\n", level, log)
}

// 获取主ip
func getAgentIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		logger("ERROR", "get agent ip failed "+err.Error())
		os.Exit(1)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0]
}

// 获取当前用户用户名
func getCurrUser() string {
	currentUser, err := user.Current()
	if err != nil {
		logger("ERROR", "get current user failed "+err.Error())
	}
	return currentUser.Name
}

// 获取当前用户家目录
func getUserPath() string {
	currentUser, err := user.Current()
	if err != nil {
		logger("ERROR", "get current user home dir failed "+err.Error())
		os.Exit(1)
	}
	return currentUser.HomeDir
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
		logger("WARN", fmt.Sprintf("connect to %s failed %s", ipv4, err.Error()))
		return false
	}
	defer conn.Close()
	return conn != nil
}

// 是否有值
func isValue(v interface{}) bool {
	if f, ok := v.(string); ok {
		if f != "" {
			return true
		}
	}
	return false
}

// 组装配置必要参数
func scanParams() (serverIP string, serverPort string, agentUser string, agentDir string, agentIP string) {
	// 接受命令
	flag.StringVar(&serverIP, "s", "", "zabbix server ip,you must input server ip.")
	flag.StringVar(&serverPort, "p", "8001", "zabbix server port.")
	flag.StringVar(&agentUser, "u", "cloud", "zabbix agent user.")
	flag.StringVar(&agentDir, "d", "", "zabbix agent directory,default is current user's home directory.")
	flag.StringVar(&agentIP, "i", "", "zabbix agent ip,default is the main ipv4.")
	// 转换
	flag.Parse()

	// serverIP,必要参数
	if !isValue(serverIP) {
		logger("ERROR", "must input zabbix server ip")
		os.Exit(1)
	}
	if !isIPv4(serverIP) {
		logger("ERROR", "invalid ip")
		os.Exit(1)
	}
	// serverPort
	if !isReachable(serverIP, serverPort) {
		logger("WARN", fmt.Sprintf("%s:%s is unreachable", serverIP, serverPort))
	}
	// agentUser
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	if agentUser != getCurrUser() && agentUser == DefaultUser {
		logger("ERROR", fmt.Sprintf("switch to default user %s then install", DefaultUser))
		os.Exit(1)
	}
	if agentUser != getCurrUser() && agentUser != DefaultUser {
		logger("ERROR", "switch to the user you specified then install")
		os.Exit(1)
	}
	// agentDir
	if !isValue(agentDir) {
		agentDir = getUserPath()
	}
	// agentIP
	if !isValue(agentIP) {
		agentIP = getAgentIP()
	}
	return serverIP, serverPort, agentUser, agentDir, agentIP
}

// 下载安装包
func fetchPackage(url string, saveAbsPath string) {
	resp, err := http.Get(url)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	// 创建文件，从url中读取文件名
	filename := path.Base(url)
	logger("INFO", fmt.Sprintf("starting to download %s from %s\n", filename, url))
	out, err := os.OpenFile(saveAbsPath+filename, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	logger("INFO", fmt.Sprintf("%s was saved to %s", filename, saveAbsPath))
	logger("INFO", "Download successful")
}

/*
// 解压安装包

	func unzipPackage(fileAbsPath string, dirAbsPath string) {
		// 检查文件是否存在
		if !isFileExist(fileAbsPath) {
			logger("INFO", "package is not found,starting to download")
			fetchPackage(LinuxURL, fileAbsPath)
		}
		// 使用linux命令解压
		logger("INFO", "starting to unzip package")
		cmd := exec.Command("tar", "--directory", dirAbsPath, "-xzf", fileAbsPath)
		_, err := cmd.Output()
		if err != nil {
			logger("ERROR", "unzip archive failed "+err.Error())
			os.Exit(1)
		}
		logger("INFO", "unzip package successful")
	}
*/
func writeINI(k string, v string, fileAbsPath string) {
	cfg, err := ini.Load(fileAbsPath)
	if err != nil {
		logger("ERROR", "write config file failed "+err.Error())
		return
	}
	cfg.Section("").Key(k).SetValue(v)
	err = cfg.SaveTo(fileAbsPath)
	if err != nil {
		logger("ERROR", "save config file failed "+err.Error())
		return
	}
}

func isFileExist(fileAbsPath string) bool {
	_, err := os.Stat(fileAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// 检查进程
func checkAgentProcess() {
	c1 := exec.Command("sh", "-c", "ps -ef|grep -v grep |grep zabbix|grep -v installer")
	out, err := c1.Output()
	if err != nil {
		logger("ERROR", "run ps failed "+err.Error())
		return
	}
	logger("INFO", "run ps successful \n"+string(out))
}

// 启动zabbix agent
func startAgent(scriptAbsPath string) {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	out, err := cmd.Output()
	if err != nil {
		logger("ERROR", "start agent failed "+err.Error())
		os.Exit(1)
	}
	logger("INFO", "start agent successful \n"+string(out))
}

// 修改启动脚本
func strBuild(zabbixDirAbsPath string, fileAbsPath string) {
	args := "s#%change_basepath%#" + zabbixDirAbsPath + "#g"
	cmd := exec.Command("sed", "-i", args, fileAbsPath)
	out, err := cmd.Output()
	if err != nil {
		logger("ERROR", "modify script failed "+err.Error())
		os.Exit(1)
	}
	logger("INFO", "modify script successful \n"+string(out))
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

	// 配置路径
	packageAbsPath := filepath.Join(AgentDir, "zabbix-agentd-5.0.14-1.linux.x86_64.tar.gz")
	zabbixDirAbsPath := filepath.Join(AgentDir, "zabbix_agentd")
	zabbixScriptAbsPath := filepath.Join(zabbixDirAbsPath, "zabbix_script.sh")
	zabbixConfDirAbsPath := filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf.d")
	zabbixConfAbsPath := filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")
	zabbixPidFileAbsPath := filepath.Join(zabbixDirAbsPath, "/zabbix_agentd.pid")
	zabbixLogFileAbsPath := filepath.Join(zabbixDirAbsPath, "/zabbix_agentd.log")

	if isFileExist(packageAbsPath) {
		// 解压安装包
		// unzipPackage(AgentDir+packagePath, AgentDir)
		// err := utils.Unzip(AgentDir+packagePath, AgentDir)
		// 解压到当前文件夹
		err := utils.Untar(packageAbsPath, "")
		if err != nil {
			logger("", "ungzip failed "+err.Error())
			return
		}
	} else {
		fetchPackage(LinuxURL, packageAbsPath)
	}

	// 写入配置
	writeINI("Include", zabbixConfDirAbsPath, zabbixConfAbsPath)
	writeINI("PidFile", zabbixPidFileAbsPath, zabbixConfAbsPath)
	writeINI("LogFile", zabbixLogFileAbsPath, zabbixConfAbsPath)
	writeINI("ServerActive", ServerIP, zabbixConfAbsPath)
	writeINI("Hostname", AgentIP, zabbixConfAbsPath)

	// 修改启动脚本
	strBuild(zabbixDirAbsPath, zabbixScriptAbsPath)

	// 启动zabbix
	startAgent(zabbixScriptAbsPath)

	// 检查进程
	checkAgentProcess()
	logger("INFO", "zabbix agent installer is running done.")
}
