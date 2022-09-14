package main

import (
	"testing"
)

func TestGetRunTimeProcessList(t *testing.T) {
	pname := GetProcessName()
	for n := range pname {
		t.Log(n)
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
