package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// UnpackingFile
func UnpackingFile(src string, dst string) error {
	_, filename := filepath.Split(src)
	if strings.Contains(filename, ".zip") {
		err := UnZip(src, dst)
		if err != nil {
			return err
		}
	} else if strings.Contains(filename, ".tar.gz") {
		err := Untar(src, dst)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unknown file format")
	}
	return nil
}
