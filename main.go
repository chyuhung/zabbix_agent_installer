package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"zabbix_agent_installer/utils"
)

// Config represents the configuration.
type Config struct {
	ServerIP    string
	ServerPort  string
	AgentIP     string
	AgentUser   string
	AgentDir    string
	PackageName string
	PackageURL  string
	OSType      string
	OSArch      string
}

type PathConfig struct {
	PackageAbsPath         string
	ZabbixAgentDirAbsPath  string
	ZabbixAgentAbsPath     string
	ZabbixAgentConfAbsPath string
}

var (
	OS_TYPE = "linux"
	OS_ARCH = "amd64"
)

const (
	EXIT     = true
	CONTINUE = false
)

func ProcessPathConfig(config *Config, pathConfig *PathConfig) error {
	switch config.OSType {
	case "linux":
		pathConfig.ZabbixAgentDirAbsPath = filepath.Join(config.AgentDir, "zabbix_agentd")
		pathConfig.ZabbixAgentAbsPath = filepath.Join(pathConfig.ZabbixAgentDirAbsPath, "zabbix_script.sh")
		pathConfig.ZabbixAgentConfAbsPath = filepath.Join(pathConfig.ZabbixAgentDirAbsPath, "/etc/zabbix_agentd.conf")
	case "windows":
		pathConfig.ZabbixAgentDirAbsPath = filepath.Join(config.AgentDir, "zabbix")
		pathConfig.ZabbixAgentAbsPath = filepath.Join(pathConfig.ZabbixAgentDirAbsPath, "bin", "zabbix_agentd.exe")
		pathConfig.ZabbixAgentConfAbsPath = filepath.Join(pathConfig.ZabbixAgentDirAbsPath, "conf", "zabbix_agentd.conf")
	}
	fileInfo, err := os.Stat(pathConfig.ZabbixAgentDirAbsPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(pathConfig.ZabbixAgentDirAbsPath, os.ModePerm)
		return err
	}
	// Check the dir
	fileMode := fileInfo.Mode()
	if fileMode.IsDir() {
		dir, err := os.ReadDir(pathConfig.ZabbixAgentDirAbsPath)
		checkError(err, EXIT)
		// if OS type is windows, stop the process.
		if len(dir) != 0 && config.OSType == "windows" {
			// Stop all zabbix agent
			_, err = RunWinCommand("taskkill", "/F", "/IM", "zabbix_agentd.exe", "/T")
			if err != nil {
				return fmt.Errorf("path %s already in use", pathConfig.ZabbixAgentDirAbsPath)
			}
		}
	}
	return nil
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

// checkError prints an error message and exit if the exit is true
func checkError(err error, exit bool) {
	if err != nil {
		Logger("ERROR", err.Error())
		if exit {
			os.Exit(1)
		}
	}
}

// unpackingPackage
func unpackingPackage(config *Config, pathConfig *PathConfig) error {
	packageAbsPath := pathConfig.PackageAbsPath
	agentDir := config.AgentDir
	packageName := config.PackageName
	if strings.Contains(packageName, ".zip") {
		err := utils.UnZip(packageAbsPath, agentDir)
		if err != nil {
			return err
		}
	} else if strings.Contains(packageName, ".tar.gz") {
		err := utils.Untar(packageAbsPath, agentDir)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unknown package format")
	}
	return nil
}

func writeConfig(config *Config, pathConfig *PathConfig) error {
	zabbixDirAbsPath := pathConfig.ZabbixAgentDirAbsPath
	zabbixConfAbsPath := pathConfig.ZabbixAgentConfAbsPath
	serverIP := config.ServerIP
	agentIP := config.AgentIP
	switch config.OSType {
	case "linux":
		confArgsMap := make(map[string]string, 3)
		confArgsMap["%change_basepath%"] = zabbixDirAbsPath
		confArgsMap["%change_serverip%"] = serverIP
		confArgsMap["%change_hostname%"] = agentIP
		Logger("INFO", "starting to modify the zabbix agent conf...")
		err := ReplaceString(zabbixConfAbsPath, confArgsMap)
		if err != nil {
			return err
		}
	case "windows":
		reMap := map[*regexp.Regexp]string{regexp.MustCompile(`.*ServerActive=.*`): "ServerActive=" + serverIP,
			regexp.MustCompile(`.*Hostname=.*`): "Hostname=" + agentIP,
		}
		f, err := os.OpenFile(zabbixConfAbsPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		all, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		result, err := RewriteLines(all, reMap)
		if err != nil {
			return err
		}
		// Write to temp file
		tempFilePath := filepath.Join(zabbixConfAbsPath + RandStringBytes(6))
		ft, err := os.OpenFile(tempFilePath, os.O_CREATE|(os.O_RDWR|os.O_TRUNC), os.ModePerm)
		if err != nil {
			return err
		}
		_, err = ft.Write(result)
		if err != nil {
			return err
		}
		// Close src file
		err = f.Close()
		if err != nil {
			return err
		}
		// Close temp file
		err = ft.Close()
		if err != nil {
			return err
		}
		// Remove src file,move new file to srr
		err = os.Remove(zabbixConfAbsPath)
		if err != nil {
			return err
		}
		err = os.Rename(tempFilePath, zabbixConfAbsPath)
		if err != nil {
			return err
		}
	}
	return nil
}
func startAgent(config *Config, pathConfig *PathConfig) error {
	zabbixDirAbsPath := pathConfig.ZabbixAgentDirAbsPath
	zabbixConfAbsPath := pathConfig.ZabbixAgentConfAbsPath
	zabbixAbsPath := pathConfig.ZabbixAgentAbsPath

	switch config.OSType {
	case "linux":
		// Modify the startup script
		rgsMap := make(map[string]string, 1)
		rgsMap["%change_basepath%"] = zabbixDirAbsPath
		err := ReplaceString(zabbixAbsPath, rgsMap)
		if err != nil {
			return err
		}

		// Start zabbix
		err = StartAgent(zabbixAbsPath)
		if err != nil {
			return err
		}

		// Check the process
		p := GetProcess()
		for pid, name := range p {
			if strings.Contains(name, "zabbix_agentd") {
				fmt.Printf("pid:%d, name:%s\n", pid, name)
			}
		}
		// Write the cron
		cron := "*/10 * * * * /bin/sh /home/test/zabbix_agentd/zabbix_script.sh daemon 2>&1 > /dev/null\n"
		err = WriteCrontab(cron)
		if err != nil {
			Logger("WARN", err.Error())
		}
	case "windows":
		err := os.Chdir(filepath.Join(zabbixDirAbsPath, "\\bin\\"))
		if err != nil {
			return err
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
	return nil
}

func main() {
	var err error
	var config = &Config{}
	var pathConfig = &PathConfig{}
	// Read the OS Info
	err = ReadOSInfo(config)
	checkError(err, EXIT)
	// Read the configuration
	ReadConfig(config)
	// Process configuration
	err = ProcessConfig(config)
	checkError(err, EXIT)

	// Check the package
	pathConfig.PackageAbsPath = filepath.Join(config.AgentDir, config.PackageName)
	// Unpacking the package
	err = unpackingPackage(config, pathConfig)
	checkError(err, EXIT)

	// Write configuration
	err = writeConfig(config, pathConfig)
	checkError(err, EXIT)
	// Start zabbix agent
	err = startAgent(config, pathConfig)
	checkError(err, EXIT)
	Logger("INFO", "the zabbix agent installer is running done.")
}
