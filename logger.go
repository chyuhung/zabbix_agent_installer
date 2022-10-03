package main

import (
	"fmt"
	"os"
	"strings"
)

// ReplaceOthers Replace the "\n","\r","\t" to " "
func ReplaceOthers(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// Logger Log messages display
func Logger(level string, messages ...string) {
	result := ""
	level = ReplaceOthers(level)
	if level == "" {
		level = "SYSTEM"
	}
	// Replace '\n' to ' '
	for i := range messages {
		result += ReplaceOthers(messages[i]) + " "
	}
	fmt.Printf("[%s] %s\n", level, result)
}

// checkError prints an error message and exit if the exit is true
func checkError(err error, exit bool) {
	if err != nil {
		Logger("ERROR", err.Error())
		if exit {
			os.Exit(1)
		}
	}
}
