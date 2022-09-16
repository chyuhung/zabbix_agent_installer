package main

import (
	"fmt"
	"os/exec"
)

// Start zabbix agent
func StartAgent(scriptAbsPath string) error {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

// Check zabbix agent process
func ShowAgentProcess() error {
	c2 := exec.Command("sh", "-c", "ps -ef|grep -E 'UID|zabbix' |grep -Ev 'installer|grep'")
	out, err := c2.Output()
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}
