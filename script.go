package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/axgle/mahonia"
	"os/exec"
)

// StartAgent Start zabbix agent
func StartAgent(scriptAbsPath string) error {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

// ShowAgentProcess Check zabbix agent process
func ShowAgentProcess() error {
	c2 := exec.Command("sh", "-c", "ps -ef|grep -E 'UID|zabbix' |grep -Ev 'installer|grep'")
	out, err := c2.Output()
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}

// RunWinCommand windows run command
func RunWinCommand(name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmdArgs := []string{"/C", name}
	for i := range args {
		cmdArgs = append(cmdArgs, args[i])
	}
	cmd := exec.Command("cmd.exe", cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	// GBK
	enc := mahonia.NewEncoder("gbk")
	outGBK := enc.ConvertString(stdout.String())
	errGBK := enc.ConvertString(stderr.String())
	if err != nil {
		return "", errors.New(fmt.Sprint(err.Error(), errGBK))
	}
	return outGBK, nil
}
