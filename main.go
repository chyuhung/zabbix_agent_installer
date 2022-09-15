package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"zabbix_agent_installer/mylog"
	"zabbix_agent_installer/mynet"
	"zabbix_agent_installer/myos"
	"zabbix_agent_installer/mystring"
	"zabbix_agent_installer/myuser"
	"zabbix_agent_installer/script"
	"zabbix_agent_installer/utils"
)

var (
	DefaultUser = "cloud"
	ServerIP    string
	ServerPort  string
	AgentUser   string
	AgentDir    string
	AgentIP     string
)

// 组装配置必要参数
func ScanParams() (server string, port string, user string, dir string, agent string) {
	// 接受命令
	flag.StringVar(&server, "s", "", "zabbix server ip.")
	flag.StringVar(&port, "p", "8001", "zabbix server port.")
	flag.StringVar(&user, "u", "cloud", "zabbix agent user.")
	flag.StringVar(&dir, "d", "", "zabbix agent directory,default is current user's home directory.")
	flag.StringVar(&agent, "i", "", "zabbix agent ip,default is host's main ip.")
	// 转换
	flag.Parse()
	// serverIP,必要参数
	if mystring.IsEmptyString(server) {
		mylog.Logger("ERROR", "must input zabbix server ip")
		os.Exit(1)
	} else if !mynet.IsIPv4(server) {
		mylog.Logger("ERROR", "invalid server ip")
		os.Exit(1)
	}

	// agentUser
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	currUser, err := myuser.GetCurrentUser()
	if err != nil {
		mylog.Logger("ERROR", "get current user failed "+err.Error())
		os.Exit(1)
	}
	if currUser != user {
		if currUser == DefaultUser {
			mylog.Logger("ERROR", fmt.Sprintf("switch to default user %s then install", DefaultUser))
			os.Exit(1)
		} else {
			mylog.Logger("ERROR", fmt.Sprintf("switch to the user %s then install", user))
			os.Exit(1)
		}
	}
	// agentDir
	if mystring.IsEmptyString(dir) {
		dir, err = myuser.GetUserHomePath()
		if err != nil {
			mylog.Logger("ERROR", "get current user home dir failed "+err.Error())
			os.Exit(1)
		}
	} else {
		// 格式化成标准写法
		dir = filepath.Join(dir)
	}

	// agentIP
	if mystring.IsEmptyString(agent) {
		agent, err = mynet.GetMainIP()
		if err != nil {
			mylog.Logger("ERROR", "get agent ip failed "+err.Error())
			os.Exit(1)
		}
	} else if !mynet.IsIPv4(agent) {
		mylog.Logger("ERROR", "invalid agent ip")
		os.Exit(1)
	}
	// serverPort
	if mynet.IsUnreachable(server, port) {
		mylog.Logger("WARN", fmt.Sprintf("connect to %s:%s failed", server, port))
	} else {
		mylog.Logger("INFO", fmt.Sprintf("connect to %s:%s successful", server, port))
	}

	return server, port, user, dir, agent
}

// GetZabbixAgentLink returns the zabbix agent link
func GetZabbixAgentLink(links []string) string {
	var zaLinks []string
	// 筛选包含关键词zabbix-agent 的链接
	for i := range links {
		if mystring.IsContainsOr(links[i], []string{"zabbix-agent", "zabbix_agent"}) {
			zaLinks = append(zaLinks, links[i])
		}
	}
	// 系统类型
	ot := runtime.GOOS
	// 架构类型
	oa := runtime.GOARCH
	var avaLinks []string

	switch ot {
	case "windows":
		for i := range zaLinks {
			if mystring.IsContainsOr(links[i], []string{"amd64"}) && mystring.IsContainsAnd(zaLinks[i], []string{"win"}) {
				avaLinks = append(avaLinks, zaLinks[i])
			} else {
				mylog.Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	case "linux":
		for i := range zaLinks {
			if oa == "amd64" {
				if mystring.IsContainsOr(zaLinks[i], []string{"amd64", "x86_64"}) && mystring.IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else if oa == "386" {
				if mystring.IsContainsOr(zaLinks[i], []string{"386"}) && mystring.IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else {
				mylog.Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	default:
		mylog.Logger("ERROR", fmt.Sprintf("unknown OS type:%s", ot))
	}
	mylog.Logger("INFO", "get links done")
	return avaLinks[len(avaLinks)-1]
}

// 筛选zabbix安装包名称
func GetZabbixAgentPackageName(filenames []string) (string, error) {
	var avaFilenames []string
	for _, filename := range filenames {
		if mystring.IsContainsAnd(filename, []string{"zabbix", "agent"}) && mystring.IsContainsOr(filename, []string{".tar.gz", ".zip"}) {
			switch runtime.GOOS {
			case "linux":
				if strings.Contains(filename, "linux") {
					avaFilenames = append(avaFilenames, filename)
				}
			case "windows":
				if strings.Contains(filename, "win") {
					avaFilenames = append(avaFilenames, filename)
				}
			default:
				mylog.Logger("ERROR", fmt.Sprintf("unknown os type: %s", runtime.GOOS))
			}
		}
	}
	if len(avaFilenames) == 0 {
		return "", fmt.Errorf("no package found")
	}
	return avaFilenames[len(avaFilenames)-1], nil
}

func main() {
	LinuxURL := "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_linux/"
	WinURL := "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_windows/"
	url := "http://10.191.101.254/zabbix-agent/"

	// 获取关键参数
	ServerIP, ServerPort, AgentUser, AgentDir, AgentIP := ScanParams()
	// 输出配置信息
	mylog.Logger("INFO", fmt.Sprintf("ServerIP:%s", ServerIP))
	mylog.Logger("INFO", fmt.Sprintf("ServerPort:%s", ServerPort))
	mylog.Logger("INFO", fmt.Sprintf("AgentUser:%s", AgentUser))
	mylog.Logger("INFO", fmt.Sprintf("AgentDir:%s", AgentDir))
	mylog.Logger("INFO", fmt.Sprintf("AgentIP:%s", AgentIP))
	// 检查安装包
	filenames, err := myos.GetFileNames(AgentDir)
	if err != nil {
		mylog.Logger("", err.Error())
		os.Exit(1)
	}
	packageName, err := GetZabbixAgentPackageName(filenames)
	if err != nil {
		mylog.Logger("", err.Error())
		// 目录下无可用安装包
		mylog.Logger("INFO", "no package found,starting to download...")
		mylog.Logger("INFO", fmt.Sprintf("os type: %s", runtime.GOOS))
		switch runtime.GOOS {
		case "linux":
			mylog.Logger("INFO", "linux is supported")
			url = LinuxURL
		case "windows":
			mylog.Logger("INFO", "windows is supported")
			url = WinURL
		default:
			mylog.Logger("ERROR", "unknown platform")
			os.Exit(1)
		}
		url = "http://10.191.101.254/zabbix-agent/"
		// 获取链接
		mylog.Logger("INFO", fmt.Sprintf("url: %s", url))
		URLs, err := mynet.GetLinks(url)
		if err != nil {
			mylog.Logger("", err.Error())
			os.Exit(1)
		}
		zaLink := GetZabbixAgentLink(URLs)
		fmt.Println("zaLink:", zaLink)

		// 下载安装包,保存在agentDir
		packageName = mynet.DownloadPackage(zaLink, AgentDir)
		mylog.Logger("INFO", fmt.Sprintf("package name: %s", packageName))

	}
	// 配置路径
	packageAbsPath := filepath.Join(AgentDir, packageName)
	zabbixDirAbsPath := filepath.Join(AgentDir, "zabbix_agentd")
	zabbixScriptAbsPath := filepath.Join(zabbixDirAbsPath, "zabbix_script.sh")
	zabbixConfAbsPath := filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")

	// 解压安装包,解压到当前文件夹
	mylog.Logger("INFO", fmt.Sprintf("starting untar %s", packageAbsPath))
	err = utils.Untar(packageAbsPath, AgentDir)
	if err != nil {
		mylog.Logger("", "ungzip failed "+err.Error())
		return
	}
	mylog.Logger("INFO", fmt.Sprintf("untar %s successful", packageAbsPath))

	// 写入配置
	confArgsMap := make(map[string]string, 3)
	confArgsMap["%change_basepath%"] = zabbixDirAbsPath
	confArgsMap["%change_serverip%"] = ServerIP
	confArgsMap["%change_hostname%"] = AgentIP
	mylog.Logger("INFO", "starting to modify zabbix agent conf")
	err = mystring.ReplaceString(zabbixConfAbsPath, confArgsMap)
	if err != nil {
		mylog.Logger("ERROR", err.Error())
	}
	mylog.Logger("INFO", "modify zabbix agent conf successful")

	// 修改启动脚本
	scriptArgsMap := make(map[string]string, 1)
	scriptArgsMap["%change_basepath%"] = zabbixDirAbsPath
	mylog.Logger("INFO", "starting to modify zabbix agent script")
	err = mystring.ReplaceString(zabbixScriptAbsPath, scriptArgsMap)
	if err != nil {
		mylog.Logger("ERROR", err.Error())
	}
	mylog.Logger("INFO", "modify zabbix agent script successful")

	// 启动zabbix
	script.StartAgent(zabbixScriptAbsPath)

	// 检查进程
	//checkAgentProcess()
	p := myos.GetProcess()
	for pid, name := range p {
		if strings.Contains(name, "zabbix_agentd") {
			fmt.Printf("pid:%d, name:%s\n", pid, name)
		}
	}
	mylog.Logger("INFO", "zabbix agent installer is running done.")
}
