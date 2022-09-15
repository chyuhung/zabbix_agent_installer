package mylog

import "fmt"

// 日志打印
func Logger(level string, log string) {
	if level == "" {
		level = "SYSTEM"
	}
	fmt.Printf("[%s] %s\n", level, log)
}
