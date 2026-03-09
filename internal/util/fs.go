package util

import (
	"os"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
