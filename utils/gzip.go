package utils

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
)

// 解压
func UnGzip(src string, dst string) error {
	gzipFile, err := os.Open(src)
	if err != nil {
		return err
	}
	gzipReader, err := gzip.NewReader(gzipFile)
	if err == io.EOF {
		return err
	}
	defer gzipReader.Close()

	outfileWriter, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outfileWriter.Close()

	_, err = io.Copy(outfileWriter, gzipReader)
	if err != nil {
		return err
	}
	return nil
}

// 压缩
func Gzip(file string, gzipWriter gzip.Writer) error {
	readFile, err := os.Open(file)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(readFile)
	for {
		s, e := reader.ReadString('\n')
		if e == io.EOF {
			break
		}
		_, err := gzipWriter.Write([]byte(s))
		if err != nil {
			return err
		}
	}
	return nil
}
