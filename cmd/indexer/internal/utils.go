package internal

import (
	"log"
	"os"
	"path/filepath"
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

func CheckSums(fp string) ([]byte, error) {
	//fmt.Println(": check sums", fp)
	return []byte{}, nil
}

func HashSumFile(fp string) (uint, error) {
	return 0, nil
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
