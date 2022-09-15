package main

import (
	"fmt"
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
	result := ContainsOr("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)
	result = ContainsOr("windows", []string{"winx", "linux", "ww", "windowsxp"})
	t.Log(result)
}

func TestContainsAnd(t *testing.T) {
	result := ContainsAnd("windows", []string{"win", "wi", "w", "windows"})
	t.Log(result)
	result = ContainsAnd("windows", []string{"win", "linux", "w", "windows"})
	t.Log(result)

}

func TestCheckProcess(t *testing.T) {
	go func() {
		for {
			fmt.Println("Checking")
			time.Sleep(1 * time.Second)
		}
	}()
	checkAgentProcess()

	t.Log("check ok")
}
