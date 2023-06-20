package common

import (
	"fmt"
	"os"
)

// IsFileBigger returns true if file size is bigger than 4GB
func IsFileBigger(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("Couldn't get info of file %v to upload. Here's why: %v\n", filePath, err)
	}
	fileSize := float64(fileInfo.Size())
	gb := float64(1024 * 1024 * 1024)
	if fileSize/gb >= 1.0 {
		return true, nil
	}
	return false, nil
}
