package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
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

// RewriteLine replace rows when the row matches re
func RewriteLine(lines []byte, re *regexp.Regexp, s string) ([]byte, error) {
	br := bytes.NewReader(lines)
	b := bufio.NewReader(br)
	for {
		line, err := b.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if re.MatchString(line) {
			if !strings.Contains(s, "\n") {
				s += "\n"
			}
			line = s
		}
	}
	return lines, nil
}

func RewriteLines(lines []byte, reMap map[*regexp.Regexp]string) ([]byte, error) {
	var result []byte
	br := bytes.NewReader(lines)
	b := bufio.NewReader(br)
	for {
		line, err := b.ReadString('\n')
		if err == io.EOF && line == "" {
			break
		} else if err != nil && line == "" {
			return nil, err
		}
		for k, v := range reMap {
			if k.MatchString(line) {
				line = v
				break
			}
		}
		if !strings.Contains(line, "\n") {
			line += "\n"
		}
		result = append(result[:], line...)
	}
	return result, nil
}

// IsContainsAnd once s not contains the one of ss , return false
func IsContainsAnd(s string, ss []string) bool {
	for i := range ss {
		if !strings.Contains(s, ss[i]) {
			return false
		}
	}
	return true
}

// IsContainsOr if s contains one of ss, return true
func IsContainsOr(s string, ss []string) bool {
	for i := range ss {
		if strings.Contains(s, ss[i]) {
			return true
		}
	}
	return false
}
