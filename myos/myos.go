package myos

import (
	"io/ioutil"
	"os"

	"github.com/shirou/gopsutil/process"
)

// 获取路径下文件名称,忽略文件夹
func GetFileNames(absPath string) ([]string, error) {
	var myFiles []string
	files, err := ioutil.ReadDir(absPath)
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
	_, err := os.Stat(fileAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
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
