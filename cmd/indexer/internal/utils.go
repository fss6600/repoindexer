package internal

import (
	"fmt"
	"os"
)

// проверяет наличие файла на диске
func FileExists(fp string) bool {
	_, err := os.Stat(fp)
	if  err == nil {
		return true
	}
	return false
}


// возвращает данные IP,.. инициатора работ в репозитории
func TaskOwnerInfo() []byte {
	return []byte("127.0.0.1") //todo: добавить информацию о подключении
}


func CheckSums(fp string) ([]byte, error) {
	fmt.Println(": check sums", fp)
	return []byte{'e',1,'a',5,5,'d',2,'f'}, nil
}


