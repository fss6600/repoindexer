package utils

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileExists проверяет наличие файла на диске
func FileExists(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

// TaskOwnerInfo возвращает данные IP,.. инициатора работ в репозитории
func TaskOwnerInfo() []byte {
	return []byte("127.0.0.1") // TODO: добавить информацию о подключении
}

// HashSum возвращает строку контрольной суммы переданной строки
func HashSum(sd string) string {
	r := strings.NewReader(sd)
	h := sha1.New()
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
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatalf(errMsg, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DirList передает в канал наименования папок в указанной директории
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

// CheckError проверяет на наличии ошибки
func CheckError(str string, err *error) {
	if *err != nil {
		panic(fmt.Errorf("%v %v", str, *err))
	}
}

// CheckPanic обработка вызова panic в любой части программы
func CheckPanic(debug bool) {
	if !debug {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}

}

// UserAccept проверяет ответ пользователя
func UserAccept(msg string) bool {
	scanner := bufio.NewScanner(os.Stdin)
	for i := 0; i < 3; i++ {
		fmt.Print(msg + ". Продолжить? (y/N): ")
		scanner.Scan()
		txt := scanner.Text()
		if len(txt) == 0 {
			return false
		} else if txt[0] == 'n' || txt[0] == 'N' {
			return false
		} else if txt[0] == 'y' || txt[0] == 'Y' {
			return true
		} else {
			continue
		}
	}
	return false
}

// ReadFromJSONFile читает данные из JSON файла
func ReadFromJSONFile(fp string, v *interface{}) error {
	buf, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}
