package main

import "fmt"

// Log printing
func Logger(level string, log string) {
	if level == "" {
		level = "SYSTEM"
	}
	fmt.Printf("[%s] %s\n", level, log)
}
