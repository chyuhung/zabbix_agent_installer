package main

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func IsEmptyString(v interface{}) bool {
	if f, ok := v.(string); ok {
		if f == "" {
			return true
		}
	}
	return false
}

// ReplaceString edits the given file,replacing all k with v.
func ReplaceString(filePath string, args map[string]string) error {
	tempFileAbsPath := filePath + ".temp"
	fi, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err = fi.Close(); err != nil {
			panic(err)
		}
	}()

	fo, err := os.OpenFile(tempFileAbsPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		if err = fo.Close(); err != nil {
			panic(err)
		}
	}()
	br := bufio.NewReader(fi)
	bw := bufio.NewWriter(fo)
	for {
		var newline string
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		for k, v := range args { // Replace each k with v
			newline = strings.ReplaceAll(line, k, v)
			line = newline
		}
		_, err = bw.WriteString(newline + "\n")
		if err != nil {
			return err
		}
	}
	// Write to a file
	err = bw.Flush()
	if err != nil {
		return err
	}
	// Remove the old file
	err = os.Remove(filePath)
	if err != nil {
		return err
	} else { // Rename the file
		err = os.Rename(tempFileAbsPath, filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// once s not contains the one of ss , return false
func IsContainsAnd(s string, ss []string) bool {
	for i := range ss {
		if !strings.Contains(s, ss[i]) {
			return false
		}
	}
	return true
}

// if s contains one of ss, return true
func IsContainsOr(s string, ss []string) bool {
	for i := range ss {
		if strings.Contains(s, ss[i]) {
			return true
		}
	}
	return false
}
