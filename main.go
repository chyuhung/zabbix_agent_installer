package main

import (
	"bufio"
	"bytes"
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
	OSType      = "linux"
	OSArch      = "amd64"
)

// ScanParams Read the parameters from stdin
func ScanParams() (server string, port string, user string, dir string, agent string, packageURL string) {
	// Receive the command
	flag.StringVar(&server, "s", "", "zabbix server ip.")
	flag.StringVar(&port, "p", "8001", "zabbix server port.")
	flag.StringVar(&agent, "i", "", "zabbix agent ip,default is host's main ip.")
	flag.StringVar(&packageURL, "l", "", "zabbix agent package URL.")
	switch OSType {
	case "linux":
		flag.StringVar(&user, "u", "cloud", "zabbix agent user.")
		flag.StringVar(&dir, "d", "", "zabbix agent directory,default is current user's home directory.")
	case "windows":
		flag.StringVar(&user, "u", "", "zabbix agent user.")
		flag.StringVar(&dir, "d", "c:\\", "zabbix agent directory.")
	}
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
	currentUser, err := GetCurrentUser() // test in WinServer2008sp2: WIN-0SH02HNMDMU\Administrator
	if err != nil {
		Logger("ERROR", "get current user failed "+err.Error())
		os.Exit(1)
	}
	if user != "" && currentUser != user {
		if strings.Contains(currentUser, DefaultUser) { // Linux "cloud",Windows "Administrator"
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

	return server, port, user, dir, agent, packageURL
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

// GetZabbixAgentPackageName Filter zabbix installation package names
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
			}
		}
	}
	if len(avaFilenames) == 0 {
		return "", fmt.Errorf("no package found")
	}
	return avaFilenames[len(avaFilenames)-1], nil
}

// NewCronFile Edit the crontab file
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

// RandStringBytes Generate rand string
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

// NewCronTempFile Generate a crontab path
func NewCronTempFile() (absPath string) {
	randString := RandStringBytes(6)
	return filepath.Join("/tmp/", "crontab."+randString)
}

func WriteCrontab(cron string) error {
	// Get the source cron
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output()
	f := bytes.NewReader(output)
	b := bufio.NewReader(f)
	isCronExists := 0
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
			isCronExists = 1
		}
	}
	// If the source cron file contains the cron
	if isCronExists == 1 {
		return fmt.Errorf("crontab already exists")
	}
	// New zabbix_agentd crontab
	dstCronFileAbsPath, err := NewCronFile(string(output) + cron)
	if err != nil {
		return err
	}
	// Rewrite the crontab
	cmd = exec.Command("crontab", dstCronFileAbsPath)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	// Remove the temp crontab file
	err = os.Remove(dstCronFileAbsPath)
	if err != nil {
		return err
	}
	return nil
}
func main() {
	PackageDirURL := "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_linux/"
	// System type
	OSType = runtime.GOOS
	// Architecture Type
	OSArch = runtime.GOARCH
	if OSType == "" || OSArch == "" {
		Logger("ERROR", "get OS info failed.")
		os.Exit(1)
	}
	switch OSType {
	case "windows":
		DefaultUser = "Administrator"
		PackageDirURL = "http://10.191.22.9:8001/software/zabbix-4.0/zabbix_agentd_windows/"
	default:
		Logger("ERROR", "OS type not supported.")
		os.Exit(1)
	}

	// Gets the key parameters
	ServerIP, ServerPort, AgentUser, AgentDir, AgentIP, PackageURL := ScanParams()
	// Output configuration information
	Logger("INFO", "ServerIP:", ServerIP, "ServerPort:", ServerPort, "AgentUser:", AgentUser, "AgentDir:", AgentDir, "AgentIP:", AgentIP, "PackageURL", PackageURL)

	// Check the package
	// Get all filenames of current dir
	filenames, err := GetFileNames(AgentDir)
	if err != nil {
		Logger("", err.Error())
		os.Exit(1)
	}
	Logger("WARN", "get filenames successfully.")
	// Check if there have package name with zabbix agent
	Logger("INFO", "starting to get the zabbix agent package name.")
	packageName, err := GetZabbixAgentPackageName(filenames)
	// if no package found,ready to download from url
	if err != nil {
		Logger("WARN", err.Error())
		// There are no installation packages available in the directory
		Logger("INFO", "starting to search package from URL...")
		// Test URL
		//PackageDirURL = "http://10.191.101.254/zabbix-agent/"
		// The link
		Logger("INFO", fmt.Sprintf("default package dir: %s", PackageDirURL))
		Logger("INFO", "starting to download...")
		URLs, err := GetLinks(PackageDirURL)
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
		if PackageURL == "" {
			PackageURL = GetZabbixAgentLink(URLs)
		}
		Logger("INFO", fmt.Sprintf("get the zabbix package link: %s", PackageURL))

		// Download the installation package and save it in agentDir
		Logger("INFO", "Downloading the zabbix package ...")
		packageName, err = DownloadPackage(PackageURL, AgentDir)
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
	}
	Logger("INFO", fmt.Sprintf("get the package name is %s", packageName))

	// Configure the path
	packageAbsPath := filepath.Join(AgentDir, packageName)
	var zabbixDirAbsPath, zabbixAbsPath, zabbixConfAbsPath string
	switch OSType {
	case "linux":
		zabbixDirAbsPath = filepath.Join(AgentDir, "zabbix_agentd")
		zabbixAbsPath = filepath.Join(zabbixDirAbsPath, "zabbix_script.sh")
		zabbixConfAbsPath = filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")
	case "windows":
		zabbixDirAbsPath = filepath.Join(AgentDir, "zabbix")
		zabbixAbsPath = filepath.Join(zabbixDirAbsPath, "bin", "zabbix_agentd.exe")
		zabbixConfAbsPath = filepath.Join(zabbixDirAbsPath, "conf", "zabbix_agentd.conf")
		info, err := os.Stat(zabbixDirAbsPath)
		// Check the dir
		fm := info.Mode()
		if fm.IsRegular() {
			Logger("ERROR", fmt.Sprintf("path %s already in use.", zabbixDirAbsPath))
			os.Exit(1)
		} else if fm.IsDir() {
			// Stop all zabbix agent
			_, err = RunWinCommand("taskkill", "/F", "/IM", "zabbix_agentd.exe", "/T")
			if err != nil {
				Logger("ERROR", "stop zabbix agent failed.", err.Error())
				Logger("INFO", "the directory may be occupied by other programs")
				os.Exit(1)
			} else {
				Logger("INFO", "stop zabbix agent successfully.")
			}
		}
		if err != nil {
			if os.IsNotExist(err) {
				err := os.MkdirAll(zabbixDirAbsPath, os.ModePerm)
				if err != nil {
					Logger("ERROR", "mkdir failed."+err.Error())
				}
			}
		}
	}

	// Unzip the installation package and extract it to the current folder
	Logger("INFO", fmt.Sprintf("starting unpacking %s", packageAbsPath))
	if strings.Contains(packageName, ".zip") {
		err = utils.UnZip(packageAbsPath, AgentDir)
		if err != nil {
			Logger("ERROR", "UnZip failed."+err.Error())
			os.Exit(1)
		}
	} else if strings.Contains(packageName, ".tar.gz") {
		err = utils.Untar(packageAbsPath, AgentDir)
		if err != nil {
			Logger("ERROR", "UnGzip failed."+err.Error())
			os.Exit(1)
		}
	} else {
		Logger("ERROR", "unknown format."+err.Error())
	}
	Logger("INFO", fmt.Sprintf("unpacking %s successfully.", packageAbsPath))

	// Write configuration
	switch OSType {
	case "linux":
		confArgsMap := make(map[string]string, 3)
		confArgsMap["%change_basepath%"] = zabbixDirAbsPath
		confArgsMap["%change_serverip%"] = ServerIP
		confArgsMap["%change_hostname%"] = AgentIP
		Logger("INFO", "starting to modify the zabbix agent conf...")
		err = ReplaceString(zabbixConfAbsPath, confArgsMap)
		if err != nil {
			Logger("", "replace string failed."+err.Error())
			os.Exit(1)
		}
	case "windows":
		reMap := map[*regexp.Regexp]string{regexp.MustCompile(`.*ServerActive=.*`): "ServerActive=" + ServerIP,
			regexp.MustCompile(`.*Hostname=.*`): "Hostname=" + AgentIP,
		}
		f, err := os.OpenFile(zabbixConfAbsPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			Logger("ERROR", "open file failed."+err.Error())
			os.Exit(1)
		}
		all, err := io.ReadAll(f)
		if err != nil {
			Logger("ERROR", "read all failed."+err.Error())
			os.Exit(1)
		}
		result, err := RewriteLines(all, reMap)
		if err != nil {
			Logger("ERROR", "rewrite lines failed."+err.Error())
			os.Exit(1)
		}
		// Write to temp file
		tempFilePath := filepath.Join(zabbixConfAbsPath + RandStringBytes(6))
		ft, err := os.OpenFile(tempFilePath, os.O_CREATE|(os.O_RDWR|os.O_TRUNC), os.ModePerm)
		_, err = ft.Write(result)
		if err != nil {
			Logger("ERROR", "write result failed."+err.Error())
			os.Exit(1)
		}
		// Close src file
		err = f.Close()
		if err != nil {
			Logger("ERROR", "close file failed."+err.Error())
			return
		}
		// Close temp file
		err = ft.Close()
		if err != nil {
			Logger("ERROR", "close temp failed."+err.Error())
			return
		}
		// Remove src file,move new file to srr
		err = os.Remove(zabbixConfAbsPath)
		if err != nil {
			Logger("ERROR", "remove file failed."+err.Error())
			os.Exit(1)
		}
		err = os.Rename(tempFilePath, zabbixConfAbsPath)
		if err != nil {
			Logger("ERROR", "rename temp file failed."+err.Error())
			os.Exit(1)
		}
	}
	Logger("INFO", "modify the zabbix agent conf successfully.")

	// Start zabbix agent
	switch OSType {
	case "linux":
		// Modify the startup script
		rgsMap := make(map[string]string, 1)
		rgsMap["%change_basepath%"] = zabbixDirAbsPath
		Logger("INFO", "starting to modify the zabbix agent script...")
		err = ReplaceString(zabbixAbsPath, rgsMap)
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
		Logger("INFO", "modify the zabbix agent script successfully.")

		// Start zabbix
		Logger("INFO", "starting to start the zabbix agent...")
		err = StartAgent(zabbixAbsPath)
		if err != nil {
			Logger("ERROR", "start the zabbix agent failed."+err.Error())
			os.Exit(1)
		}
		Logger("INFO", "starting the zabbix agent successfully.")

		// Check the process
		p := GetProcess()
		for pid, name := range p {
			if strings.Contains(name, "zabbix_agentd") {
				fmt.Printf("pid:%d, name:%s\n", pid, name)
			}
		}
		// Write the cron
		Logger("INFO", "starting write cron...")
		cron := "*/10 * * * * /bin/sh /home/test/zabbix_agentd/zabbix_script.sh daemon 2>&1 > /dev/null\n"
		err = WriteCrontab(cron)
		if err != nil {
			Logger("ERROR", err.Error())
		}
		Logger("INFO", "write crontab successfully.")
	case "windows":
		err := os.Chdir(filepath.Join(zabbixDirAbsPath, "\\bin\\"))
		if err != nil {
			Logger("ERROR", "change current dir failed."+err.Error())
			os.Exit(1)
		}

		// Uninstall zabbix agent
		_, err = RunWinCommand(zabbixAbsPath, "-c", zabbixConfAbsPath, "-d")
		if err != nil {
			Logger("ERROR", "uninstall zabbix agent failed.", err.Error())
		} else {
			Logger("INFO", "uninstall zabbix agent successfully.")
		}
		// Install zabbix agent
		_, err = RunWinCommand(zabbixAbsPath, "-c", zabbixConfAbsPath, "-i")
		if err != nil {
			Logger("ERROR", "install zabbix agent failed.", err.Error())
		} else {
			Logger("INFO", "install zabbix agent successfully.")
		}
		// Start zabbix agent
		_, err = RunWinCommand(zabbixAbsPath, "-c", zabbixConfAbsPath, "-s")
		if err != nil {
			Logger("ERROR", "start zabbix agent failed.", err.Error())
		} else {
			Logger("INFO", "start zabbix agent successfully.")
		}
	}

	Logger("INFO", "the zabbix agent installer is running done.")
}
