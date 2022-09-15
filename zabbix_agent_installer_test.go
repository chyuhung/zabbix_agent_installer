package main

import (
	"fmt"
	"testing"
	"time"
	"zabbix_agent_installer/myos"
	"zabbix_agent_installer/mystring"
	"zabbix_agent_installer/script"
)

func TestGetRunTimeProcessList(t *testing.T) {
	p := myos.GetProcess()
	for k, v := range p {
		t.Log("pid:", k, "name:", v)
	}
}

func TestContainsOr(t *testing.T) {
	result := mystring.IsContainsOr("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)
	result = mystring.IsContainsOr("windows", []string{"winx", "linux", "ww", "windowsxp"})
	t.Log(result)
}

func TestContainsAnd(t *testing.T) {
	result := mystring.IsContainsAnd("windows", []string{"win", "wi", "w", "windows"})
	t.Log(result)
	result = mystring.IsContainsAnd("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)

}

func TestCheckProcess(t *testing.T) {
	go func() {
		for {
			fmt.Println("Checking")
			time.Sleep(1 * time.Second)
		}
	}()
	script.ShowAgentProcess()

	t.Log("check ok")
}
