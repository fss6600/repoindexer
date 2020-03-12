package utils

import (
	"crypto"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// проверяет наличие файла на диске
func FileExists(fp string) bool {
	_, err := os.Stat(fp)
	if err == nil {
		return true
	}
	return false
}

// возвращает данные IP,.. инициатора работ в репозитории
func TaskOwnerInfo() []byte {
	return []byte("127.0.0.1") //todo: добавить информацию о подключении
}

func HashSum(sd string) string {
	r := strings.NewReader(sd)
	h := crypto.SHA1.New()
	_, _ = io.Copy(h, r)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// HashSumFile вычисляет контрольную сумму файла по алгоритму SHA1
func HashSumFile(fp string) (string, error) {
	errMsg := "file hash sum calc error: %v"
	f, err := os.Open(fp)
	if err != nil {
		log.Fatalf(errMsg, err)
	}
	defer f.Close()
	h := crypto.SHA1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatalf(errMsg, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

//...
func DirList(fp string, dirs chan<- string) {
	fList, err := filepath.Glob(filepath.Join(fp, "*"))
	if err != nil {
		log.Fatalf("DirList error: %v", err)
	}
	for _, d := range fList {
		res, err := os.Stat(d)
		if err != nil {
			log.Fatalf("DirList error: %v", err)
		}
		if res.IsDir() {
			dirs <- filepath.Base(d)
		}
	}
	close(dirs)
}
