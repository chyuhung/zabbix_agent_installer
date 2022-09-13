package main

import (
	"testing"
)

func TestGetOSType(t *testing.T) {
	OSType := GetOSType()
	t.Log(OSType)
}

func TestGetOSArch(t *testing.T) {
	OSArch := GetOSArch()
	t.Log(OSArch)
}

func TestGetRunTimeProcessList(t *testing.T) {
	pname := GetProcessName()
	for n := range pname {
		t.Log(n)
	}
}
