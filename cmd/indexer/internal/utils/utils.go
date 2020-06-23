package utils

import (
	"bufio"
	"compress/gzip"
	"crypto"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	// "strings"
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

func WriteGzip(jsonData []byte, fp string) error {
	errMsg := fmt.Sprintf(":WriteGzip: %v")
	indexFile, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	zw := gzip.NewWriter(indexFile)
	defer func() {
		_ = zw.Close()
		_ = indexFile.Close()
	}()

	_, err = zw.Write(jsonData)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func WriteGzipHash(fp, hash string) error {
	errMsg := fmt.Sprintf(":writeGzipHash: %v")
	indexFileHash, err := os.Create(fp + ".sha1")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	defer func() {
		if err := indexFileHash.Close(); err != nil {
			panic(err)
		}
	}()

	_, err = indexFileHash.Write([]byte(hash))
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

//...
func CheckError(str string, err *error) {
	if *err != nil {
		panic(fmt.Errorf("%v %v", str, *err))
	}
}

// обработка вызова panic в любой части программы
func CheckPanic(debug bool) {
	if !debug {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}

}

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

// Рекурсивно обходит указанную папку и возвращает имена файлов в указанный канал или ошибки в соответствующий канал
func DirWalk(root string, fpCh chan<- string, erCh chan<- error) {
	err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
		if er != nil {
			return fmt.Errorf("не найден пакет: %q\n", fp)
		}
		if info.IsDir() { // skip directory
			return nil
		}
		fp, _ = filepath.Rel(root, fp) // trim base Path repopath/packname
		fpCh <- fp
		return nil
	})
	if err != nil {
		erCh <- err
		return
	}
	close(fpCh)
}
