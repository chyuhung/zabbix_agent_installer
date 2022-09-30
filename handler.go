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
	flag.StringVar(&config.AgentIP, "i", "", "zabbix agent ip,default is host's main ip.")
	flag.StringVar(&config.PackageURL, "l", "", "zabbix agent package URL.")
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
	/*
		user := config.AgentUser
		currentUser, err := GetCurrentUser() // test in WinServer2008sp2: WIN-0SH02HNMDMU\Administrator.
		if err != nil {
			return fmt.Errorf("get current user failed")
		}
		if user != "" && user != currentUser {
			if strings.Contains(currentUser, DEFAULT_USER) { // Linux "cloud",Windows "Administrator"
				return fmt.Errorf("switch to default user %s then install", DEFAULT_USER)
			} else {
				return fmt.Errorf("switch to the user %s then install", config.AgentUser)
			}
		}
		config.AgentUser = currentUser
		return nil
	*/
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

	/*
		dir := config.AgentDir
		if IsEmptyString(dir) {
			dir, err := GetUserHomePath()
			if err != nil {
				return errors.New("get current user home dir failed." + err.Error())
				os.Exit(1)
			}
			config.AgentDir = dir
		} else {
			dir = filepath.Join(dir)
			Logger("INFO", fmt.Sprintf("get current user home dir is %s", dir))
			config.AgentDir = dir
		}*/
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
	return nil
}

func ProcessConfig(config *Config) {
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
	// Check package URL
	err = packageURLHandler(config)
	checkError(err, EXIT)
}
