package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Start zabbix agent
func StartAgent(scriptAbsPath string) {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	_, err := cmd.Output()
	if err != nil {
		Logger("ERROR", "start agent failed "+err.Error())
		os.Exit(1)
	}
	Logger("INFO", "start agent successful")
}

// Check zabbix agent process
func ShowAgentProcess() {
	c2 := exec.Command("sh", "-c", "ps -ef|grep -E 'UID|zabbix' |grep -Ev 'installer|grep'")
	out, err := c2.Output()
	if err != nil {
		Logger("ERROR", "run ps failed "+err.Error())
		return
	}
	Logger("INFO", "run ps successful")
	fmt.Print(string(out))
}
