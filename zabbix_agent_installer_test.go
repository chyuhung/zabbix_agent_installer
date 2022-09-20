package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"testing"
	"time"
)

func TestGetRunTimeProcessList(t *testing.T) {
	p := GetProcess()
	for k, v := range p {
		t.Log("pid:", k, "name:", v)
	}
}

func TestContainsOr(t *testing.T) {
	result := IsContainsOr("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)
	result = IsContainsOr("windows", []string{"winx", "linux", "ww", "windowsxp"})
	t.Log(result)
}

func TestContainsAnd(t *testing.T) {
	result := IsContainsAnd("windows", []string{"win", "wi", "w", "windows"})
	t.Log(result)
	result = IsContainsAnd("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)

}

func TestCheckProcess(t *testing.T) {
	go func() {
		for {
			fmt.Println("Checking")
			time.Sleep(1 * time.Second)
		}
	}()
	err := ShowAgentProcess()
	if err != nil {
		return
	}

	t.Log("check ok")
}

func TestWriteCrontab(t *testing.T) {
	// Get the source cron
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output()
	t.Logf(string(output))

	f := bytes.NewReader(output)
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			t.Log(err.Error())
		}
		t.Logf(line)
	}
}

func TestWriteCrontab1(t *testing.T) {

	cron := "*/10 * * * * /bin/sh /home/test/zabbix_agentd/zabbix_script.sh daemon 2>&1 > /dev/null\n"
	err := WriteCrontab(cron)
	if err != nil {
		t.Logf(err.Error())
	}
}

// Windows test
func TestGetCurrentUser(t *testing.T) {
	user, err := GetCurrentUser()
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf(user)
}
func TestGetUserHomePath(t *testing.T) {
	userHomePath, err := GetUserHomePath()
	if err != nil {
		t.Logf(err.Error())
	}
	t.Logf(userHomePath)
}
