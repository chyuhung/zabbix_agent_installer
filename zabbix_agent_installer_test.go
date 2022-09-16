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
	ShowAgentProcess()

	t.Log("check ok")
}
