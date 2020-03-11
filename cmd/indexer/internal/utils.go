package internal

import (
	"os"
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
