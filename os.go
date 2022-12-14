package main

import (
	"os"

	"github.com/shirou/gopsutil/process"
)

// GetFileNames Gets the filename under path, ignoring the folder
func GetFileNames(absPath string) ([]string, error) {
	var myFiles []string
	files, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if !file.IsDir() {
			myFiles = append(myFiles, file.Name())
		}
	}
	return myFiles, nil
}

// IsFileNotExist returns true if the given file exists,otherwise returns false.
func IsFileNotExist(fileAbsPath string) bool {
	fileInfo, err := os.Stat(fileAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
	}
	if fileInfo.IsDir() {
		return false
	}
	return false
}

// GetProcess returns the list of runtime processes
func GetProcess() map[int32]string {
	p := make(map[int32]string, 30)
	pids, _ := process.Pids()
	for _, pid := range pids {
		pn, _ := process.NewProcess(pid)
		name, _ := pn.Name()
		p[pid] = name
	}
	return p
}
