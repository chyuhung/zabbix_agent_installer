package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"time"
)

// zabbix agent username
// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
var (
	Username = "cloud"
)

type agent struct {
	serverIP  string
	port      string
	username  string
	directory string
}

func NewAgent() *agent {
	return &agent{}
}

// 配置安装用户信息
func (a *agent) setParams(serverIP string, port string, username string, directory string) {
	a.serverIP = serverIP
	a.port = port
	a.username = username
	a.directory = directory
}

// 帮助信息
func helpInfo() {
}
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
func getUsername() string {
	currentUser, err := user.Current()
	if err != nil {
		logger("", err.Error())
	}
	return currentUser.Name
}

// 获取当前用户家目录
func getHomeDir() string {
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

func isIPv4(ipv4 string) bool {
	ip := net.ParseIP(ipv4)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	return ip != nil
}

func isServerReachable(ipv4 string, port string) bool {
	addr := net.JoinHostPort(ipv4, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		logger("", err.Error())
		return false
	}
	defer conn.Close()
	return conn != nil
}

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
	if isServerReachable(serverIP, port) {
		logger("INFO", fmt.Sprintf("%s:%s is reachable", serverIP, port))
	} else {
		logger("INFO", fmt.Sprintf("%s:%s is unreachable", serverIP, port))

	}

	// username
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	if username != getUsername() && username == Username {
		logger("ERROR", "请切换到默认用户再安装")
		os.Exit(1)
	}
	if username != getUsername() && username != Username {
		logger("ERROR", "请切换到指定用户再安装")
		os.Exit(1)
	}
	// directory
	if directory == "" {
		directory = getHomeDir()
	}
	return serverIP, port, username, directory
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
	fmt.Println(agent.username)
	fmt.Println(agent.directory)
}
