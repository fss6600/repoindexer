package handler

import (
	"bufio"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	// errMsg := "file hash sum calc error: %v"
	f, err := os.Open(fp)
	if err != nil {
		return "", &internalError{
			Text:   fmt.Sprintf("ошибка подсчета контрольной суммы файла %s", fp),
			Caller: "HashSumFile",
			Err:    err,
		}
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", &internalError{
			Text:   fmt.Sprintf("ошибка подсчета контрольной суммы файла %s", fp),
			Caller: "HashSumFile",
			Err:    err,
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DirList передает в канал наименования папок в указанной директории
func DirList(fp string, dirs chan<- string) {
	fList, err := filepath.Glob(filepath.Join(fp, "*"))
	if err != nil {
		log.Fatal(&internalError{
			Text:   "ошибка получения списка пакетов",
			Caller: "DirList::Glob",
			Err:    err,
		})
	}
	for _, d := range fList {
		res, err := os.Stat(d)
		if err != nil {
			log.Fatal(&internalError{
				Text:   "ошибка получения данных пакета",
				Caller: "DirList::Stat",
				Err:    err,
			})
		}
		if res.IsDir() {
			dirs <- filepath.Base(d)
		}
	}
	close(dirs)
}

// CheckPanic обработка вызова panic в любой части программы
// func CheckPanic(debug bool) {
// 	if !debug {
// 		if r := recover(); r != nil {
// 			fmt.Println(r)
// 		}
// 	}

// }

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
		return &internalError{
			Text:   "ошибка чтения файла данных",
			Caller: "ReadFromJSONFile",
			Err:    err,
		}
	}
	return json.Unmarshal(buf, v)
}

func searchExecFile(root string, regExp *regexp.Regexp) ([]string, error) {
	execFilesList := []string{}
	for fInfo := range dirWalk(root) {
		if regExp.MatchString(fInfo.Path) {
			fp, err := filepath.Rel(root, fInfo.Path)
			if err != nil {
				return nil, &internalError{
					Text:   "ошибка приведения относительного пути",
					Caller: "searchExecFile::Rel",
					Err:    err,
				}
			}
			execFilesList = append(execFilesList, fp)
		}
	}
	return execFilesList, nil
}

func selectExecFileByUser(fList []string) string {
	scanner := bufio.NewScanner(os.Stdin)
	count := len(fList)
	for {
		fmt.Printf("введите число от 1 до %d:\n", count)
		for i := 0; i < count; i++ {
			fmt.Printf("\t[%d]: '%v'\n", i+1, fList[i])
		}
		scanner.Scan()
		choice := scanner.Text()
		choiceInt, err := strconv.Atoi(choice)
		if err == nil && choiceInt != 0 && choiceInt <= count {
			return fList[choiceInt-1]
		}
	}
}

func defineExecFile(r *Repo, pack string) (string, error) {
	var (
		execFilesList []string
		execFile      string
	)
	packRoot := filepath.Join(r.Path(), pack)
	execRegEx, _ := regexp.Compile(`^.+\.exe$`)
	if execFilesList, err = searchExecFile(packRoot, execRegEx); err != nil {
		return "", err
	}

	switch len(execFilesList) {
	case 0:
		execFile = "noexec"
	case 1:
		execFile = execFilesList[0]
	default:
		fmt.Printf("Выберите исполняемый файл для пакета '%v'\n", pack)
		execFile = selectExecFileByUser(execFilesList)
	}
	return execFile, nil
}

// ShowEmptyExecFiles выводит на консоль список пакетов, для которых требуется указать исполняемый файл
func ShowEmptyExecFiles(r *Repo) {
	emptyList := r.EmptyExecFilesList()
	if len(emptyList) > 0 {
		fmt.Println("\n\tДля следующих пакетов требуется указать исполняемый файл:")
		for _, pack := range emptyList {
			fmt.Printf("\t\t%v\n", pack)
		}
		fmt.Println("\tЗапустите программу с командой 'exec check'")
	}
}

// dirWalk Рекурсивно обходит указанную папку и возвращает канал
// с данными о файлах
func dirWalk(root string) chan FileInfo {
	fInfoCh := make(chan FileInfo)
	fInfo := FileInfo{}

	go func() {
		err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
			if er != nil {
				return fmt.Errorf("не найден пакет: %q", filepath.Base(fp))
			}
			if info.IsDir() { // skip directory
				return nil
			}
			fInfo.Path = fp
			fInfo.Size = info.Size()
			fInfo.MDate = info.ModTime().UnixNano()
			fInfoCh <- fInfo
			return nil
		})
		if err != nil {
			log.Fatal(&internalError{
				Text:   "ошибка обхода директории",
				Caller: "dirWalk::goroutine",
				Err:    err,
			})
		}
		close(fInfoCh)
	}()
	return fInfoCh
}

// InitDB инициализирует файл db
func InitDB(path string) error {
	fp := pathDB(path)
	if FileExists(fp) {
		return &internalError{
			Text:   "попытка повторной инициализации",
			Caller: "InitDB",
		}
	}
	db, err := newConnection(fp)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	if _, err := db.Exec(initSQL); err != nil {
		return &internalError{
			Text:   "ошибка выполнения SQL",
			Caller: "InitDB::db.Exec::init",
			Err:    err,
		}
	}
	if _, err := db.Exec("INSERT INTO info (id, vers_major, vers_minor) VALUES (?, ?, ?);",
		1, DBVersionMajor, DBVersionMinor); err != nil {
		return &internalError{
			Text:   "ошибка выполнения SQL",
			Caller: "InitDB::db.Exec::insert",
			Err:    err,
		}
	}
	fmt.Println("Репозиторий инициализирован")
	return nil
}

// CleanForMigrate удаляет файлы БД, индекса
func CleanForMigrate(repo *Repo) error {
	for _, fp := range []string{fileDBName, IndexGZ, IndexGZ + ".sha1"} {
		fp = filepath.Join(repo.path, fp)
		if err = os.Remove(fp); err != nil {
			return &internalError{
				Text:   "ошибка удаления файла",
				Caller: "CleanForMigrate",
				Err:    err,
			}
		}
	}
	return nil
}

func newConnection(fp string) (*sql.DB, error) {
	sql, err := sql.Open("sqlite3", fp)
	return sql, &internalError{
		Text:   "ошибка открытия файла БД",
		Caller: "newConnection",
		Err:    err,
	}
}

func pathDB(repoPath string) string {
	return filepath.Join(repoPath, fileDBName)
}
