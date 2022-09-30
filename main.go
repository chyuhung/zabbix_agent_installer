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
	DEFAULT_USER = "cloud"
	OS_TYPE      = "linux"
	OS_ARCH      = "amd64"
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
				return fmt.Errorf("path %s already in use.", pathConfig.ZabbixAgentDirAbsPath)
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

	/*
		ServerIP, ServerPort, AgentUser, AgentDir, AgentIP, PackageURL, PackageName := "1", "2", "3", "4", "5", "6", "7"
		// Output configuration information
		Logger("INFO", "ServerIP:", ServerIP)
		Logger("INFO", "ServerPort:", ServerPort)
		Logger("INFO", "AgentUser:", AgentUser)
		Logger("INFO", "AgentDir:", AgentDir)
		Logger("INFO", "AgentIP:", AgentIP)
		Logger("INFO", "PackageURL:", PackageURL)
		Logger("INFO", "PackageName:", PackageName)

		// find from the URL
		if PackageName == "" {
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
			PackageName, err = GetZabbixAgentPackageName(filenames)
			// if no package found,ready to download from url
			if err != nil {
				Logger("WARN", err.Error())
				if PackageURL == "" {
					// There are no installation packages available in the directory
					Logger("INFO", "starting to search package from URL...")
					Logger("INFO", fmt.Sprintf("get the zabbix package link: %s", PackageURL))
					// Test URL
					//PackageDirURL = "http://10.191.101.254/zabbix-agent/"
					// The link
					Logger("INFO", fmt.Sprintf("default package dir: %s", PACKAGE_URL))
					Logger("INFO", "starting to download...")
					URLs, err := GetLinks(PACKAGE_URL)
					if err != nil {
						Logger("ERROR", err.Error())
						os.Exit(1)
					}
					PackageURL = GetZabbixAgentLink(URLs)
				}

				// Download the installation package and save it in agentDir
				Logger("INFO", "Downloading the zabbix package ...")
				PackageName, err = DownloadPackage(PackageURL, AgentDir)
				if err != nil {
					Logger("ERROR", err.Error())
					os.Exit(1)
				}
			}
		}
		Logger("INFO", fmt.Sprintf("get the package name is %s", PackageName))

		// Configure the path
		packageAbsPath := filepath.Join(AgentDir, PackageName)
		var zabbixDirAbsPath, zabbixAbsPath, zabbixConfAbsPath string
		switch OS_TYPE {
		case "linux":
			zabbixDirAbsPath = filepath.Join(AgentDir, "zabbix_agentd")
			zabbixAbsPath = filepath.Join(zabbixDirAbsPath, "zabbix_script.sh")
			zabbixConfAbsPath = filepath.Join(zabbixDirAbsPath, "/etc/zabbix_agentd.conf")
		case "windows":
			zabbixDirAbsPath = filepath.Join(AgentDir, "zabbix")
			zabbixAbsPath = filepath.Join(zabbixDirAbsPath, "bin", "zabbix_agentd.exe")
			zabbixConfAbsPath = filepath.Join(zabbixDirAbsPath, "conf", "zabbix_agentd.conf")
			info, err := os.Stat(zabbixDirAbsPath)
			if err != nil {
				if os.IsNotExist(err) {
					err := os.MkdirAll(zabbixDirAbsPath, os.ModePerm)
					if err != nil {
						Logger("ERROR", "mkdir failed.", err.Error())
					}
				} else {
					Logger("ERROR", err.Error())
					os.Exit(1)
				}
			}
			// Check the dir
			infoMode := info.Mode()
			if infoMode.IsDir() {
				dir, _ := os.ReadDir(zabbixDirAbsPath)
				if len(dir) != 0 {
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
			} else {
				Logger("ERROR", fmt.Sprintf("path %s already in use.", zabbixDirAbsPath))
				os.Exit(1)
			}
		}*/

	// Unzip the installation package and extract it to the current folder
	Logger("INFO", fmt.Sprintf("starting unpacking %s", packageAbsPath))
	if strings.Contains(PackageName, ".zip") {
		err := utils.UnZip(packageAbsPath, AgentDir)
		if err != nil {
			Logger("ERROR", "unzip failed.", err.Error())
			os.Exit(1)
		}
	} else if strings.Contains(PackageName, ".tar.gz") {
		err := utils.Untar(packageAbsPath, AgentDir)
		if err != nil {
			Logger("ERROR", "unGzip failed.", err.Error())
			os.Exit(1)
		}
	} else {
		Logger("ERROR", "unknown package format. check the package URL.")
		os.Exit(1)
	}
	Logger("INFO", fmt.Sprintf("unpacking %s successfully.", packageAbsPath))

	// Write configuration
	switch OS_TYPE {
	case "linux":
		confArgsMap := make(map[string]string, 3)
		confArgsMap["%change_basepath%"] = zabbixDirAbsPath
		confArgsMap["%change_serverip%"] = ServerIP
		confArgsMap["%change_hostname%"] = AgentIP
		Logger("INFO", "starting to modify the zabbix agent conf...")
		err := ReplaceString(zabbixConfAbsPath, confArgsMap)
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
		if err != nil {
			Logger("ERROR", err.Error())
			os.Exit(1)
		}
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
	switch OS_TYPE {
	case "linux":
		// Modify the startup script
		rgsMap := make(map[string]string, 1)
		rgsMap["%change_basepath%"] = zabbixDirAbsPath
		Logger("INFO", "starting to modify the zabbix agent script...")
		err := ReplaceString(zabbixAbsPath, rgsMap)
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
