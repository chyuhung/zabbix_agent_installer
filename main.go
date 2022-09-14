package main

import (
	"bufio"
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
	"runtime"
	"strings"

	"time"
	"zabbix_agent_installer/utils"

	"github.com/shirou/gopsutil/process"
	"golang.org/x/net/html"
)

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

// 获取当前用户用户名
func getCurrUser() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.Username, nil
}

// 获取当前用户家目录
func getUserPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.HomeDir, nil
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
func isUnreach(ipv4 string, port string) bool {
	addr := net.JoinHostPort(ipv4, port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return true
	}
	defer conn.Close()
	return false
}

// 是否有值
func isEmptyStr(v interface{}) bool {
	if f, ok := v.(string); ok {
		if f == "" {
			return true
		}
	}
	return false
}

// IsFileExist returns true if the given file exists,otherwise returns false.
func IsFileExist(fileAbsPath string) bool {
	_, err := os.Stat(fileAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// ReplaceWords edits the given file,replacing all k with v.
func ReplaceWords(filePath string, args map[string]string) error {
	tempFileAbsPath := filePath + ".temp"
	fi, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err = fi.Close(); err != nil {
			panic(err)
		}
	}()

	fo, err := os.OpenFile(tempFileAbsPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		if err = fo.Close(); err != nil {
			panic(err)
		}
	}()
	br := bufio.NewReader(fi)
	bw := bufio.NewWriter(fo)
	for {
		var newline string
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		for k, v := range args { //逐个替换kv
			newline = strings.ReplaceAll(line, k, v)
			line = newline
		}
		_, err = bw.WriteString(newline + "\n")
		if err != nil {
			return err
		}
	}
	// 写入文件
	err = bw.Flush()
	if err != nil {
		return err
	}
	// 移除旧文件
	err = os.Remove(filePath)
	if err != nil {
		return err
	} else { // 重命名新文件
		err = os.Rename(tempFileAbsPath, filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetMainIP gets the IP address of the host.
func GetMainIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0], nil
}

// 组装配置必要参数
func scanParams() (server string, port string, user string, dir string, agent string) {
	// 接受命令
	flag.StringVar(&server, "s", "", "zabbix server ip.")
	flag.StringVar(&port, "p", "8001", "zabbix server port.")
	flag.StringVar(&user, "u", "cloud", "zabbix agent user.")
	flag.StringVar(&dir, "d", "", "zabbix agent directory,default is current user's home directory.")
	flag.StringVar(&agent, "i", "", "zabbix agent ip,default is host's main ip.")
	// 转换
	flag.Parse()
	// serverIP,必要参数
	if isEmptyStr(server) {
		logger("ERROR", "must input zabbix server ip")
		os.Exit(1)
	} else if !isIPv4(server) {
		logger("ERROR", "invalid server ip")
		os.Exit(1)
	}

	// agentUser
	// 如果用户不指定用户，则默认使用cloud用户，如果cloud不存在，则panic，cloud存在则检查当前是否为cloud,是则安装，不是则panic
	// 如果用户指定了用户，则检查用户是否存在，检查当前是否为指定用户，存在且是当前用户则安装，否则panic
	currUser, err := getCurrUser()
	if err != nil {
		logger("ERROR", "get current user failed "+err.Error())
		os.Exit(1)
	}
	if currUser != user {
		if currUser == DefaultUser {
			logger("ERROR", fmt.Sprintf("switch to default user %s then install", DefaultUser))
			os.Exit(1)
		} else {
			logger("ERROR", fmt.Sprintf("switch to the user %s then install", user))
			os.Exit(1)
		}
	}
	// agentDir
	if isEmptyStr(dir) {
		dir, err = getUserPath()
		if err != nil {
			logger("ERROR", "get current user home dir failed "+err.Error())
			os.Exit(1)
		}
	} else {
		// 格式化成标准写法
		dir = filepath.Join(dir)
	}

	// agentIP
	if isEmptyStr(agent) {
		agent, err = GetMainIP()
		if err != nil {
			logger("ERROR", "get agent ip failed "+err.Error())
			os.Exit(1)
		}
	} else if !isIPv4(agent) {
		logger("ERROR", "invalid agent ip")
		os.Exit(1)
	}
	// serverPort
	if isUnreach(server, port) {
		logger("WARN", fmt.Sprintf("connect to %s:%s failed", server, port))
	} else {
		logger("INFO", fmt.Sprintf("connect to %s:%s successful", server, port))
	}

	return server, port, user, dir, agent
}

// 下载安装包
func fetchPackage(url string, saveAbsPath string) {
	resp, err := http.Get(url)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logger("ERROR", err.Error())
			os.Exit(1)
		}
	}()
	// 创建文件，从url中读取文件名
	filename := path.Base(url)
	logger("INFO", fmt.Sprintf("starting to download %s from %s", filename, url))
	out, err := os.OpenFile(saveAbsPath+filename, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			logger("ERROR", err.Error())
			os.Exit(1)
		}
	}()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	logger("INFO", fmt.Sprintf("%s was saved to %s", filename, saveAbsPath))
	logger("INFO", "Download successful")
}

// 检查进程
func checkAgentProcess() {

	c2 := exec.Command("sh", "-c", "ps -ef|grep -E 'UID|zabbix' |grep -Ev 'installer|grep'")
	out, err := c2.Output()
	if err != nil {
		logger("ERROR", "run ps failed "+err.Error())
		return
	}
	logger("INFO", "run ps successful")
	fmt.Print(string(out))
}

// 启动zabbix agent
func startAgent(scriptAbsPath string) {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	_, err := cmd.Output()
	if err != nil {
		logger("ERROR", "start agent failed "+err.Error())
		os.Exit(1)
	}
	logger("INFO", "start agent successful")
}

// GetProcessName returns the list of runtime processes
func GetProcessName() (pname []string) {
	pids, _ := process.Pids()
	for _, pid := range pids {
		pn, _ := process.NewProcess(pid)
		name, _ := pn.Name()
		pname = append(pname, name)
	}
	return pname
}

// source:http://www.codebaoku.com/it-go/it-go-168428.html
func visit(links []string, n *html.Node) []string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				links = append(links, a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = visit(links, c)
	}
	return links
}

// GetURLLinks returns the name of the package
func GetURLLinks(url string) []string {
	var links []string
	resp, err := http.Get(url)
	if err != nil {
		logger("ERROR", err.Error())
		return nil
	}
	defer resp.Body.Close()
	doc, _ := html.Parse(resp.Body)
	for _, link := range visit(nil, doc) {
		links = append(links, url+link)
	}
	return links
}

// once s not contains the one of ss , return false
func ContainsAnd(s string, ss []string) bool {
	for i := range ss {
		if !strings.Contains(s, ss[i]) {
			return false
		}
	}
	return true
}

// if s contains one of ss, return true
func ContainsOr(s string, ss []string) bool {
	for i := range ss {
		if strings.Contains(s, ss[i]) {
			return true
		}
	}
	return false
}

// GetZabbixAgentLink returns the zabbix agent link
func GetZabbixAgentLink(links []string) string {
	var zaLinks []string
	// 筛选包含关键词zabbix-agent 的链接
	for i := range links {
		if ContainsOr(links[i], []string{"zabbix-agent", "zabbix_agent"}) {
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
			if ContainsOr(links[i], []string{"win", "amd64"}) {
				avaLinks = append(avaLinks, zaLinks[i])
			}
		}
	case "linux":
		for i := range zaLinks {
			if oa == "amd64" {
				if ContainsOr(zaLinks[i], []string{"amd64", "x86_64"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else if oa == "386" {
				if strings.Contains(zaLinks[i], "386") {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			}
		}

	}
	for i := range avaLinks {
		fmt.Println(avaLinks[i])
	}
	return avaLinks[len(avaLinks)-1]
}

/*
package list

linix
zabbix-agent-4.0.12-1.el5.i386.rpm
zabbix-agent-4.0.15-1.el8.x86_64.rpm
zabbix-agent-4.0.15-1.el12.x86_64.rpm
zabbix-agentd-4.0.32-1.linux.aarch64.tar.gz

windows
zabbix_agents-4.0.7-win-amd64.zip
zabbix_agentd-5.0.15-windows-amd64.zip
*/

func main() {
	url := "https://mirrors.tuna.tsinghua.edu.cn/zabbix/zabbix/4.0/rhel/6/x86_64/"
	URLs := GetURLLinks(url)
	zaLink := GetZabbixAgentLink(URLs)
	fmt.Println("zaLink:", zaLink)
	fetchPackage(zaLink, "/home/cloud/zabbix_agent_installer/tmp/")
	return

	/*
		pname := GetProcessName()
		for _, p := range pname {
			fmt.Println(p)
		}
		return
	*/
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
	zabbixConfAbsPath := filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")

	if IsFileExist(packageAbsPath) {
		// 解压安装包,解压到当前文件夹
		logger("INFO", fmt.Sprintf("starting untar %s", packageAbsPath))
		err := utils.Untar(packageAbsPath, AgentDir)
		if err != nil {
			logger("", "ungzip failed "+err.Error())
			return
		}
		logger("INFO", fmt.Sprintf("untar %s successful", packageAbsPath))
	} else {
		// 下载安装包
		fetchPackage(zaLink, packageAbsPath)
	}

	// 写入配置
	confArgsMap := make(map[string]string, 3)
	confArgsMap["%change_basepath%"] = zabbixDirAbsPath
	confArgsMap["%change_serverip%"] = ServerIP
	confArgsMap["%change_hostname%"] = AgentIP
	logger("INFO", "starting to modify zabbix agent conf")
	err := ReplaceWords(zabbixConfAbsPath, confArgsMap)
	if err != nil {
		logger("ERROR", err.Error())
	}
	logger("INFO", "modify zabbix agent conf successful")

	// 修改启动脚本
	scriptArgsMap := make(map[string]string, 1)
	scriptArgsMap["%change_basepath%"] = zabbixDirAbsPath
	logger("INFO", "starting to modify zabbix agent script")
	err = ReplaceWords(zabbixScriptAbsPath, scriptArgsMap)
	if err != nil {
		logger("ERROR", err.Error())
	}
	logger("INFO", "modify zabbix agent script successful")

	// 启动zabbix
	startAgent(zabbixScriptAbsPath)

	// 检查进程
	checkAgentProcess()
	logger("INFO", "zabbix agent installer is running done.")
}
