package main

import (
	"fmt"
	"strings"
)

// ReplaceOthers Replace the "\n","\r","\t" to " "
func ReplaceOthers(s string) string {
	strings.ReplaceAll(s, "\n", " ")
	strings.ReplaceAll(s, "\t", " ")
	strings.ReplaceAll(s, "\r", " ")
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
