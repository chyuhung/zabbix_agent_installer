package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ReadOSInfo reads the runtime information.
func ReadOSInfo(config *Config) {
	config.OSType = runtime.GOOS
	config.OSArch = runtime.GOARCH
}

// ReadConfig reads the configuration from the stdin.
func ReadConfig(config *Config) {
	// Receive the command
	flag.StringVar(&config.ServerIP, "s", "", "zabbix server ip.")
	flag.StringVar(&config.ServerPort, "p", "8001", "zabbix server port.")
	flag.StringVar(&config.AgentIP, "i", "", "zabbix agent ip,default is host's main ip.")
	flag.StringVar(&config.PackageURL, "l", "", "zabbix agent package URL. Download from the URL if no package in the dir.")
	flag.StringVar(&config.PackageName, "f", "", "zabbix agent package name.")
	switch config.OSType {
	case "linux":
		flag.StringVar(&config.AgentUser, "u", "cloud", "zabbix agent user.")
		flag.StringVar(&config.AgentDir, "d", "", "zabbix agent directory,default is current user's home directory.")
	case "windows":
		flag.StringVar(&config.AgentUser, "u", "", "zabbix agent user.")
		flag.StringVar(&config.AgentDir, "d", "c:\\", "zabbix agent directory.")
	}
	flag.Parse()
}

// serverIPHandler processes the ServerIP.
func serverIPHandler(config *Config) {
	if IsEmptyString(config.ServerIP) {
		Logger("ERROR", "must input the zabbix server ip")
		os.Exit(1)
	} else if !IsIPv4(config.ServerIP) {
		Logger("ERROR", "invalid server ip")
		os.Exit(1)
	}
}

// agentUserHandler processes the AgentUser.
// If you do not specify a user, you use the cloud user by default,
// if the cloud does not exist, then panic, cloud exists to check whether it is currently cloud,
// yes is installed, not panic;
// If a user is specified, check if the user exists, check whether the user is currently the specified user,
// exists and is the current user is installed, otherwise panic.
func agentUserHandler(config *Config) {
	user := config.AgentUser
	currentUser, err := GetCurrentUser() // test in WinServer2008sp2: WIN-0SH02HNMDMU\Administrator.
	if err != nil {
		Logger("ERROR", "get current user failed "+err.Error())
		os.Exit(1)
	}
	if user != "" && user != currentUser {
		if strings.Contains(currentUser, DEFAULT_USER) { // Linux "cloud",Windows "Administrator"
			Logger("ERROR", fmt.Sprintf("switch to default user %s then install", DEFAULT_USER))
			os.Exit(1)
		} else {
			Logger("ERROR", fmt.Sprintf("switch to the user %s then install", config.AgentUser))
			os.Exit(1)
		}
	}
	config.AgentUser = currentUser
}

// agentDirHandler processes the AgentDir
func agentDirHandler(config *Config) {
	dir := config.AgentDir
	if IsEmptyString(dir) {
		dir, err := GetUserHomePath()
		if err != nil {
			Logger("ERROR", "get current user home dir failed "+err.Error())
			os.Exit(1)
		}
		config.AgentDir = dir
	} else {
		dir = filepath.Join(dir)
		Logger("INFO", fmt.Sprintf("get current user home dir is %s", dir))
		config.AgentDir = dir
	}
}

// agentIPHandler processes the AgentIP
func agentIPHandler(config *Config) {
	agent := config.AgentIP
	if IsEmptyString(agent) {
		agent, err := GetMainIP()
		if err != nil {
			Logger("ERROR", "get agent ip failed "+err.Error())
			os.Exit(1)
		}
		config.AgentIP = agent
	} else if !IsIPv4(agent) {
		Logger("ERROR", "invalid agent ip")
		os.Exit(1)
	}
}

// serverPortHandler processes the ServerPort and ServerIP
func serverPortHandler(config *Config) {
	server := config.ServerIP
	port := config.ServerPort
	if IsUnreachable(server, port) {
		Logger("WARN", fmt.Sprintf("connect to %s:%s failed", server, port))
	} else {
		Logger("INFO", fmt.Sprintf("connect to %s:%s successfully", server, port))
	}
}

// packageURL processes the PackageURL
func packageURLHandler(config *Config) {
	packageURL := config.PackageURL
	if IsEmptyString(packageURL) {
		return
	}
	reg, err := regexp.Compile(`[a-zA-z]+://[^\s]*`)
	if err != nil {
		Logger("ERROR", err.Error())
		os.Exit(1)
	}
	if !reg.MatchString(packageURL) {
		Logger("ERROR", fmt.Sprintf("invalid package URL: %s", packageURL))
		os.Exit(1)
	}
}

func ProcessConfig(config *Config) {
	serverIPHandler(config)
	serverPortHandler(config)
	agentDirHandler(config)
	agentIPHandler(config)
	agentUserHandler(config)
	packageURLHandler(config)
}
