package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

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

// Read the parameters from stdin
func ScanParams() (server string, port string, user string, dir string, agent string) {
	// Receive the command
	flag.StringVar(&server, "s", "", "zabbix server ip.")
	flag.StringVar(&port, "p", "8001", "zabbix server port.")
	flag.StringVar(&user, "u", "cloud", "zabbix agent user.")
	flag.StringVar(&dir, "d", "", "zabbix agent directory,default is current user's home directory.")
	flag.StringVar(&agent, "i", "", "zabbix agent ip,default is host's main ip.")
	flag.Parse()
	// serverIP,Required parameters
	if IsEmptyString(server) {
		Logger("ERROR", "must input the zabbix server ip")
		os.Exit(1)
	} else if !IsIPv4(server) {
		Logger("ERROR", "invalid server ip")
		os.Exit(1)
	}

	// agentUser
	// If you do not specify a user, you use the cloud user by default, if the cloud does not exist, then panic, cloud exists to check whether it is currently cloud, yes is installed, not panic;
	// If a user is specified, check if the user exists, check whether the user is currently the specified user, exists and is the current user is installed, otherwise panic
	currentUser, err := GetCurrentUser()
	if err != nil {
		Logger("ERROR", "get current user failed "+err.Error())
		os.Exit(1)
	}
	if currentUser != user {
		if currentUser == DefaultUser {
			Logger("ERROR", fmt.Sprintf("switch to default user %s then install", DefaultUser))
			os.Exit(1)
		} else {
			Logger("ERROR", fmt.Sprintf("switch to the user %s then install", user))
			os.Exit(1)
		}
	}
	// agentDir
	if IsEmptyString(dir) {
		dir, err = GetUserHomePath()
		if err != nil {
			Logger("ERROR", "get current user home dir failed "+err.Error())
			os.Exit(1)
		}
	} else {
		dir = filepath.Join(dir)
		Logger("INFO", fmt.Sprintf("get current user home dir is %s", dir))
	}

	// agentIP
	if IsEmptyString(agent) {
		agent, err = GetMainIP()
		if err != nil {
			Logger("ERROR", "get agent ip failed "+err.Error())
			os.Exit(1)
		}
	} else if !IsIPv4(agent) {
		Logger("ERROR", "invalid agent ip")
		os.Exit(1)
	}
	// serverPort
	if IsUnreachable(server, port) {
		Logger("WARN", fmt.Sprintf("connect to %s:%s failed", server, port))
	} else {
		Logger("INFO", fmt.Sprintf("connect to %s:%s successfully", server, port))
	}

	return server, port, user, dir, agent
}

// GetZabbixAgentLink returns the zabbix agent link
func GetZabbixAgentLink(links []string) string {
	var zaLinks []string
	// Filter links that contain the keyword zabbix-agent or zabbix_agent
	for i := range links {
		if IsContainsOr(links[i], []string{"zabbix-agent", "zabbix_agent"}) {
			zaLinks = append(zaLinks, links[i])
		}
	}
	// OS type,windows or linux
	ot := runtime.GOOS
	// System architecture
	oa := runtime.GOARCH
	var avaLinks []string

	switch ot {
	case "windows":
		for i := range zaLinks {
			if IsContainsOr(links[i], []string{"amd64"}) && IsContainsAnd(zaLinks[i], []string{"win"}) {
				avaLinks = append(avaLinks, zaLinks[i])
			} else {
				Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	case "linux":
		for i := range zaLinks {
			if oa == "amd64" {
				if IsContainsOr(zaLinks[i], []string{"amd64", "x86_64"}) && IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else if oa == "386" {
				if IsContainsOr(zaLinks[i], []string{"386"}) && IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else {
				Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	default:
		Logger("ERROR", fmt.Sprintf("unknown OS type:%s", ot))
	}
	return avaLinks[len(avaLinks)-1]
}

// Filter zabbix installation package names
func GetZabbixAgentPackageName(filenames []string) (string, error) {
	var avaFilenames []string
	for _, filename := range filenames {
		if IsContainsAnd(filename, []string{"zabbix", "agent"}) && IsContainsOr(filename, []string{".tar.gz", ".zip"}) {
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
				Logger("ERROR", fmt.Sprintf("unknown os type: %s", runtime.GOOS))
			}
		}
	}
	if len(avaFilenames) == 0 {
		return "", fmt.Errorf("no package found")
	}
	return avaFilenames[len(avaFilenames)-1], nil
}

// Edit the crontab file
func NewCronFile(cron string) (string, error) {
	cronAbsPath := filepath.Join(NewCronTempFile(), "")
	f, err := os.OpenFile(cronAbsPath, os.O_CREATE|(os.O_RDWR|os.O_TRUNC), 0644)
	defer func() {
		err = f.Close()
		if err != nil {
			fmt.Print(err.Error())
			return
		}
	}()
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(cron)
	if err != nil {
		return "", err
	}
	return cronAbsPath, nil
}

// Generate a crontab expression
func GetCrontabExpression(time string, exec string, expression string) (string, error) {
	cron := time + " " + exec + " " + expression + "\n"
	if time != "" && exec != "" && expression != "" {
		return cron, nil
	} else {
		return "", fmt.Errorf("invalid crontab format:%s", cron)
	}
}

// Generate rand string
func RandStringBytes(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	size := len(letterBytes)
	for i := range b {
		w, _ := rand.Int(rand.Reader, big.NewInt(int64(size)))
		b[i] = letterBytes[w.Int64()]
	}
	return string(b)
}

// Generate a crontab path
func NewCronTempFile() (absPath string) {
	rand := RandStringBytes(6)
	return filepath.Join("/tmp/", "crontab."+rand)
}

func WriteCrontab(cron string, user string) error {
	// Get the source cron
	cmd := exec.Command("crontab", "-l")
	if user != "" {
		cmd = exec.Command("crontab", "-u", user, "-l")
	}
	output, err := cmd.Output()
	if !strings.Contains(string(output), "no crontab for") && err != nil {
		return err
	}
	// Write the output to the temp file
	srcCronFileAbsPath, err := NewCronFile(string(output))
	if err != nil {
		return err
	}
	// Open the temp file, check whether the cron exists
	f, err := os.Open(srcCronFileAbsPath)
	defer func() {
		err = f.Close()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}()
	if err != nil {
		return err
	}
	b := bufio.NewReader(f)
	isContais := 0
	for {
		line, err := b.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		// Check if the temp file contains the zabbix agent crontab
		pattern := regexp.MustCompile(`^[^#].*zabbix_agentd`)
		if pattern.MatchString(line) {
			isContais = 1
		}
	}
	// Delete the temp file
	err = os.Remove(srcCronFileAbsPath)
	if err != nil {
		return err
	}
	// If the source cron file contains the cron
	if isContais == 1 {
		return nil
	}
	// New zabbix_agentd crontab
	dstCronFileAbsPath, err := NewCronFile(string(output) + cron)
	if err != nil {
		return err
	}
	// Rewrite the crontab
	cmd = exec.Command("crontab", dstCronFileAbsPath)
	if user != "" {
		cmd = exec.Command("crontab", "-u", user, dstCronFileAbsPath)
	}
	result, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	// Remove the temp crontab file
	os.Remove(dstCronFileAbsPath)
	return nil
}
func main() {

	LinuxURL := "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_linux/"
	WinURL := "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_windows/"
	url := "http://10.191.101.254/zabbix-agent/"

	// Gets the key parameters
	ServerIP, ServerPort, AgentUser, AgentDir, AgentIP := ScanParams()
	// Output configuration information
	Logger("INFO", fmt.Sprintf("ServerIP:%s", ServerIP))
	Logger("INFO", fmt.Sprintf("ServerPort:%s", ServerPort))
	Logger("INFO", fmt.Sprintf("AgentUser:%s", AgentUser))
	Logger("INFO", fmt.Sprintf("AgentDir:%s", AgentDir))
	Logger("INFO", fmt.Sprintf("AgentIP:%s", AgentIP))
	// Check the package
	filenames, err := GetFileNames(AgentDir)
	if err != nil {
		Logger("", err.Error())
		os.Exit(1)
	}
	Logger("INFO", "get filenames successfully")
	Logger("INFO", "starting to get the zabbix agent package name")
	packageName, err := GetZabbixAgentPackageName(filenames)
	if err != nil {
		Logger("WARN", err.Error())
		// There are no installation packages available in the directory
		Logger("INFO", "starting to download...")
		Logger("INFO", fmt.Sprintf("os type: %s", runtime.GOOS))
		switch runtime.GOOS {
		case "linux":
			Logger("INFO", "linux is supported")
			url = LinuxURL
		case "windows":
			Logger("INFO", "windows is supported")
			url = WinURL
		default:
			Logger("ERROR", "unknown platform")
			os.Exit(1)
		}
		// Test url
		url = "http://10.191.101.254/zabbix-agent/"
		// Get the links
		Logger("INFO", fmt.Sprintf("url: %s", url))
		URLs, err := GetLinks(url)
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
		zaLink := GetZabbixAgentLink(URLs)
		Logger("INFO", fmt.Sprintf("get the zabbix package link: %s", zaLink))

		// Download the installation package and save it in agentDir
		Logger("INFO", "Downloading the zabbix package ...")
		packageName, err = DownloadPackage(zaLink, AgentDir)
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
	}
	Logger("INFO", fmt.Sprintf("get the package name is %s", packageName))
	// Configure the path
	packageAbsPath := filepath.Join(AgentDir, packageName)
	zabbixDirAbsPath := filepath.Join(AgentDir, "zabbix_agentd")
	zabbixAbsPath := filepath.Join(zabbixDirAbsPath, "zabbix_script.sh")
	zabbixConfAbsPath := filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")

	// Unzip the installation package and extract it to the current folder
	Logger("INFO", fmt.Sprintf("starting untar %s", packageAbsPath))
	err = utils.Untar(packageAbsPath, AgentDir)
	if err != nil {
		Logger("ERROR", "ungzip failed "+err.Error())
		os.Exit(1)
	}
	Logger("INFO", fmt.Sprintf("untar %s successfully", packageAbsPath))

	// Write configuration
	confArgsMap := make(map[string]string, 3)
	confArgsMap["%change_basepath%"] = zabbixDirAbsPath
	confArgsMap["%change_serverip%"] = ServerIP
	confArgsMap["%change_hostname%"] = AgentIP
	Logger("INFO", "starting to modify the zabbix agent conf")
	err = ReplaceString(zabbixConfAbsPath, confArgsMap)
	if err != nil {
		Logger("", err.Error())
		os.Exit(1)
	}
	Logger("INFO", "modify the zabbix agent conf successfully")

	// Modify the startup script
	rgsMap := make(map[string]string, 1)
	rgsMap["%change_basepath%"] = zabbixDirAbsPath
	Logger("INFO", "starting to modify the zabbix agent script")
	err = ReplaceString(zabbixAbsPath, rgsMap)
	if err != nil {
		Logger("ERROR", err.Error())
		os.Exit(1)
	}
	Logger("INFO", "modify the zabbix agent script successfully")

	// Start zabbix
	Logger("INFO", "starting to start the zabbix agent")
	err = StartAgent(zabbixAbsPath)
	if err != nil {
		Logger("ERROR", err.Error())
		os.Exit(1)
	}
	Logger("INFO", "starting the zabbix agent successfully")

	// Check the process
	p := GetProcess()
	for pid, name := range p {
		if strings.Contains(name, "zabbix_agentd") {
			fmt.Printf("pid:%d, name:%s\n", pid, name)
		}
	}
	// Write the cron
	Logger("INFO", "starting write cron")
	cron, err := GetCrontabExpression("*/10 * * * *", "/bin/sh", "/home/test/zabbix_agentd/zabbix_script.sh daemon 2>&1 > /dev/null")
	if err != nil {
		Logger("ERROR", err.Error())
	}
	err = WriteCrontab(cron, "")
	if err != nil {
		Logger("ERROR", err.Error())
	}
	Logger("INFO", "write crontab successfully")
	Logger("INFO", "the zabbix agent installer is running done")
}
