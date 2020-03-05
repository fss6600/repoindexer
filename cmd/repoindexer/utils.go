package main

import (
	"os"
)

// проверяет наличие файла на диске
func fileExists(fp string) bool {
	_, err := os.Stat(fp)
	if  err == nil {
		return true
	}
	return false
}


