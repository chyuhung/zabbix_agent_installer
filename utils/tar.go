package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

func ExistDir(dirname string) bool {
	fi, err := os.Stat(dirname)
	// 没有err且isDir
	// 报错存在且isDir
	return (err == nil || os.IsExist(err)) && fi.IsDir()
}

// 解压
func Untar(src string, dst string) error {
	// 打开待解压tar文件
	fr, err := os.Open(src)
	if err != nil {
		return err
	}
	// gzip解压，如未使用gzip可注释
	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()
	// tar 解压
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil // End of archive
		case err != nil:
			return err
		case hdr == nil:
			continue
		}
		// 设置保存路径为header中的name
		dstFile := filepath.Join(dst, hdr.Name)
		// 判断文件类型
		switch hdr.Typeflag {
		case tar.TypeDir: // 是目录，创建目录
			//目录是否存在，不存在则创建
			if b := ExistDir(dstFile); !b {
				// MkdirAll递归创建
				if err := os.MkdirAll(dstFile, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg: // 文件，写入
			// 创建可读写文件
			file, err := os.OpenFile(dstFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(hdr.Mode))

			if err != nil {
				return err
			}
			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
			//fmt.Printf("untar %s, char %d\n", dstFile, n)
			file.Close()
		}
	}
}
