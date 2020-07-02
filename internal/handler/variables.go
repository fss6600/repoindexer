package handler

import (
	"database/sql"
	"fmt"
	"time"
)

// internalError структура возвращаемой ошибки из handler
type internalError struct {
	Text   string // текст ошибки
	Caller string // вызвавший объект
	Err    error  // оригинальная ошибка
}

func (e *internalError) Error() (msg string) {
	if e.Caller != "" {
		msg = fmt.Sprintf("%s: %s", e.Caller, e.Text)
	} else {
		msg = e.Text
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\noriginal: %v", e.Err)
	}
	return
}

var err error

const (
	fileDBName string = "index.db"
	// IndexGZ индекс-файл
	IndexGZ string = "index.gz"
	// DBVersionMajor major ver DB
	DBVersionMajor int64 = 1
	// DBVersionMinor minor ver DB
	DBVersionMinor int64 = 4
)

// ErrAlias ошибка обработки псевдонима
type ErrAlias error

// general
const (
	fnReglament = "__REGLAMENT__"
	doPopMsg    = "\n\tВыгрузите данные в индекс-файл командой 'pop'\n"
	doIndexMsg  = "\n\tПроиндексируйте пакеты командой 'index [...pacnames]'\n"
	noChangeMsg = "Изменений нет\n"
)

// статусы пакета
const (
	PackStatusNotIndexed = iota - 1 // не индексирован
	PackStatusBlocked               // блокирован
	PackStatusActive                // активный
)

// HashedPackData структура для репрезентации данных о пакете в БД
type HashedPackData struct {
	ID    int64             `json:"-"`
	Name  string            `json:"-"`
	Alias string            `json:"alias"`
	Hash  string            `json:"phash"`
	Exec  string            `json:"execf"`
	Files map[string]string `json:"files"`
}

// RepoStData структура для сбора данных по команде status
type RepoStData struct {
	TotalCnt   int       // общее количество пакетов
	IndexedCnt int       // количество проиндексированных пакетов
	BlockedCnt int       // количество заблокированных
	IndexSize  int64     // размер индекс-файла в байтах
	DBSize     int64     // размер файла БД в байтах
	HashSize   int64     // размер хэш-файла в байтах
	IndexMDate time.Time // дата изменения индекс-файла
	DBMDate    time.Time // дата изменения файла БД
	HashMDate  time.Time // дата изменения хэш-файла
}

// ListData структура для сбора и передачи данных списка пакетов
type ListData struct {
	Status int8
	Name   string
}

// Repo объект репозитория с БД
type Repo struct {
	path        string
	disPacks    []string // список заблокированных пакетов
	actPacks    []string // список активных (актуальных) пакетов
	indPacks    []string // список проиндексированных пакетов
	db          *sql.DB
	stmtAddFile *sql.Stmt // предустановка запроса на добавление данных файла пакета в БД
	stmtDelFile *sql.Stmt // предустановка запроса на удаление данных файла пакетав БД
	stmtUpdFile *sql.Stmt // предустановка запроса на изменение данных файла пакета в БД
}

// FileInfo структура с данными о файле пакета в БД
type FileInfo struct {
	ID    int64  // package ID
	Path  string // путь файла относительно корневой папки пакета
	Size  int64  // размер файла
	MDate int64  // дата изменения
	Hash  string // контрольная сумма
}
