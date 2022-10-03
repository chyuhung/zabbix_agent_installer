package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
)

// ReadOSInfo reads the runtime information.
func ReadOSInfo(config *Config) error {
	config.OSType = runtime.GOOS
	config.OSArch = runtime.GOARCH
	switch config.OSType {
	case "linux":
	case "windows":
	default:
		return fmt.Errorf("OS type not supported")
	}
	return nil
}

// ReadConfig reads the configuration from the stdin.
func ReadConfig(config *Config) {
	// Receive the command
	flag.StringVar(&config.ServerIP, "s", "", "zabbix server ip.")
	flag.StringVar(&config.ServerPort, "p", "8001", "zabbix server port.")
	flag.StringVar(&config.AgentIP, "i", "", "zabbix agent ip. default is the main ip.")
	flag.StringVar(&config.PackageURL, "l", "", "zabbix agent package URL.")
	flag.StringVar(&config.PackageName, "f", "", "zabbix agent package name.")
	flag.StringVar(&config.AgentDir, "d", "", "zabbix agent directory. default is current dir.")
	flag.StringVar(&config.AgentUser, "u", "", "zabbix agent user. default is current user.")
	flag.Parse()
}

// serverIPHandler processes the ServerIP.
func serverIPHandler(config *Config) error {
	if config.ServerIP == "" {
		return errors.New("must input the zabbix server ip")
	} else if !IsIPv4(config.ServerIP) {
		return errors.New("invalid server ip")
	}
	return nil
}

// agentUserHandler processes the AgentUser.
// If not specify a user, use current user default,
// If user is specified,check if the current user is the specified user.
func agentUserHandler(config *Config) error {
	if os.Geteuid() == 0 {
		return errors.New("switch to normal user then install")
	}
	if os.Getuid() != -1 {
		currentUser, _ := GetCurrentUser()
		if config.AgentUser != "" && currentUser != config.AgentUser {
			return fmt.Errorf("switch to %s then install", config.AgentUser)
		}
	}
	return nil
}

// agentDirHandler processes the AgentDir
func agentDirHandler(config *Config) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	config.AgentDir = dir
	return nil
}

// agentIPHandler processes the AgentIP
func agentIPHandler(config *Config) error {
	agent := config.AgentIP
	if agent == "" {
		agent, err := GetMainIP()
		if err != nil {
			return errors.New("get agent ip failed")
		}
		config.AgentIP = agent
	} else if !IsIPv4(agent) {
		return errors.New("invalid agent ip")
	}
	return nil
}

// serverPortHandler processes the ServerPort and ServerIP
func serverPortHandler(config *Config) error {
	server := config.ServerIP
	port := config.ServerPort
	if IsUnreachable(server, port) {
		return fmt.Errorf("connect to %s:%s failed", server, port)
	}
	return nil
}

// packageNameHandler processes the package name
func packageNameHandler(config *Config) error {
	if config.PackageName == "" {
		return nil
	}
	fileInfo, err := os.Stat(config.PackageName)
	checkError(err, EXIT)
	fileMode := fileInfo.Mode()
	if fileMode.IsDir() {
		return fmt.Errorf("invalid package name: %s", config.PackageName)
	}
	return nil
}

// packageURL processes the PackageURL
func packageURLHandler(config *Config) error {
	packageURL := config.PackageURL
	if packageURL == "" {
		return nil
	}
	reg, err := regexp.Compile(`[a-zA-z]+://[^\s]*`)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	if !reg.MatchString(packageURL) {
		return fmt.Errorf("invalid package URL: %s", packageURL)
	}
	config.PackageName, err = DownloadPackage(config.PackageURL, config.AgentDir)
	checkError(err, EXIT)
	return nil
}

func ProcessConfig(config *Config) error {
	var err error
	// Check server ip
	err = serverIPHandler(config)
	checkError(err, EXIT)
	// Check server port
	err = serverPortHandler(config)
	checkError(err, CONTINUE)
	// Check server dir
	err = agentDirHandler(config)
	checkError(err, EXIT)
	// Check agent ip
	err = agentIPHandler(config)
	checkError(err, EXIT)
	// Check agent user
	err = agentUserHandler(config)
	checkError(err, EXIT)
	// Check package name
	err = packageNameHandler(config)
	checkError(err, CONTINUE)
	// Check package URL
	err = packageURLHandler(config)
	checkError(err, EXIT)
	if config.PackageName == "" && config.PackageURL == "" {
		fmt.Printf("use -f or -l to specify package URI")
	}
	return nil
}
