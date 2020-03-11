package obj

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal"

	_ "github.com/mattn/go-sqlite3"
)

const fileDBName string = "index.db"

// Repo объект репозитория с БД
type Repo struct {
	path string
	db   *sql.DB
	//FullMode      bool //?
	disPacks    []string  // список заблокированных пакетов
	actPacks    []string  // список активных (актуальных) пакетов
	stmtAddFile *sql.Stmt // предустановка запроса на добавление данных файла пакета в БД
	stmtDelFile *sql.Stmt // предустановка запроса на удаление данных файла пакетав БД
	stmtUpdFile *sql.Stmt // предустановка запроса на изменение данных файла пакета в БД
}

// Структура с данными о файле пакета в БД
type FileInfo struct {
	Id    int64
	Path  string // путь файла относительно корневой папки пакета
	Size  int64  // размер файла
	MDate int64  // дата изменения
	Hash  uint   // контрольная сумма
}

// NewRepoObj возвращает объект repoObj
func NewRepoObj(path string) *Repo {
	repo := new(Repo)
	repo.SetPath(path)
	return repo
}

//...
func (r *Repo) SetPath(path string) {
	if path == "" {
		log.Fatalln("укажите путь к репозиторию")
	}
	r.path = path
}

//...
func (r *Repo) Path() string {
	return r.path
}

// SetFullMode устанавливает режим полной индексации
//func (r *Repo) SetFullMode() {
//	r.FullMode = true
//	fmt.Println("установлен режим полной индексации")
//}

// OpenDB открывает подключение к БД
func (r *Repo) OpenDB() error {
	fp := dbPath(r.path)
	if !internal.FileExists(fp) {
		return errors.New("репозиторий не инициализирован")
	}
	db, err := newConnection(fp)
	if err != nil {
		return err
	}
	// инициализация поддержки первичных ключей в БД
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		fmt.Println("error init PRAGMA", err)
	}
	r.db = db
	return nil
}

// Close закрывает db соединение
func (r *Repo) Close() {
	if r.stmtAddFile != nil {
		_ = r.stmtAddFile.Close()
	}
	if r.stmtDelFile != nil {
		_ = r.stmtDelFile.Close()
	}
	if r.stmtUpdFile != nil {
		_ = r.stmtUpdFile.Close()
	}
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			log.Println("error close db:", err)
		}
	}
}

// PackageID возвращает ID пакета
func (r *Repo) PackageID(pack string) (id int64) {
	err := r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := r.db.Exec("INSERT INTO packages ('name', 'hash') VALUES (?, 0)", pack)
		if err != nil {
			log.Fatalf("create pack [ %v ] record in db: %v", pack, err)
		}
		id, _ = res.LastInsertId()
	}
	return
}

// ActivePacks кэширует и возвращает список пакетов в репозитории, за исключением заблокированных
func (r *Repo) ActivePacks() []string {
	if len(r.actPacks) == 0 {
		ch := make(chan string, 3)
		go internal.DirList(r.Path(), ch)
		for name := range ch {
			if r.packIsBlocked(name) {
				continue
			} else {
				r.actPacks = append(r.actPacks, name)
			}
		}
	}
	return r.actPacks
}

// DisabledPacks кэширует и возвращает список заблокированных пакетов репозитория
func (r *Repo) DisabledPacks() []string {
	if len(r.disPacks) == 0 {
		rows, err := r.db.Query("SELECT name FROM excludes;")
		if err != nil {
			log.Fatalf("error select disabled packs: %v", err)
		}
		defer rows.Close()
		var name string
		for rows.Next() {
			_ = rows.Scan(&name)
			r.disPacks = append(r.disPacks, name)
		}
	}
	return r.disPacks
}

// FilesPackRepo возвращает список файлов указанного пакета в репозитории
func (r *Repo) FilesPackRepo(pack string) ([]string, error) {
	path := filepath.Join(r.path, pack) // base Path repopath/packname
	fList := make([]string, 0, 50)      // reserve place for ~50 files
	fpCh := make(chan string)           // channel for filepath
	erCh := make(chan error)            // channel for error
	unWanted, _ := regexp.Compile("(.*[Tt]humb[s]?\\.db)|(.*~.*)")

	//go walkPack(Path, fpCh, erCh)
	go func(root string, fpCh chan<- string, erCh chan<- error) {
		err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
			if er != nil {
				// debug message: er
				return fmt.Errorf("не найден пакет: %q\n", fp)
			}
			if info.IsDir() { // skip directory
				return nil
			} else if unWanted.MatchString(fp) { // skip unwanted file
				//fmt.Println("skip unwanted:", fp) // todo add to log.debug
				return nil
			}
			fp, _ = filepath.Rel(path, fp) // trim base Path repopath/packname
			fpCh <- fp
			return nil
		})
		if err != nil {
			erCh <- err
			return
		}
		close(fpCh)
	}(path, fpCh, erCh)

	for {
		select {
		case err := <-erCh:
			return nil, err
		case fp, next := <-fpCh:
			if next {
				fList = append(fList, fp)
			} else {
				return fList, nil
			}
		}
	}
}

// FilesPackDB возвращвет список файлов пакета имеющихся в БД
func (r *Repo) FilesPackDB(id int64) ([]FileInfo, error) {
	cnt := 0
	if err := r.db.QueryRow("SELECT COUNT(*) FROM FILES WHERE package_id=?;", id).Scan(&cnt); err != nil {
		return nil, err
	}
	if cnt == 0 {
		return []FileInfo{}, nil
	}

	rows, err := r.db.Query("SELECT id, path, size, mdate, hash FROM files WHERE package_id=?;", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fdataList := make([]FileInfo, 0)
	for rows.Next() {
		fd := new(FileInfo)
		if err := rows.Scan(&fd.Id, &fd.Path, &fd.Size, &fd.MDate, &fd.Hash); err != nil {
			return nil, err
		}
		fdataList = append(fdataList, *fd)
	}
	return fdataList, nil
}

//...
func (r *Repo) packIsBlocked(name string) bool {
	for _, fn := range r.DisabledPacks() {
		if name == fn {
			return true
		}
	}
	return false
}

//...
func (r *Repo) IsActive(pack string) bool {
	for _, fp := range r.ActivePacks() {
		if fp == pack {
			return true
		}
	}
	return false
}

//...
func (r *Repo) AddFile(id int64, pack string, fPath string) error { // todo run in go
	fp := filepath.Join(r.path, pack, fPath)
	fInfo, err := os.Stat(fp)
	if err != nil {
		return err
	}
	hash, err := internal.HashSumFile(fp)
	if err != nil {
		return err
	}
	if res, err := r.stmtAddFile.Exec(id, fPath, fInfo.Size(), fInfo.ModTime().UnixNano(), hash); err != nil {
		return fmt.Errorf("stmtAddFile error: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return fmt.Errorf("stmtAddFile error: no rows added in fact")
		}
	}
	return nil
}

//..
func (r *Repo) ChangedFile(pack, fsPath string, dbData FileInfo) (bool, error) {
	fp := filepath.Join(r.path, pack, fsPath)
	fInfo, err := os.Stat(fp)
	if err != nil {
		return false, err
	}
	if fInfo.Size() == dbData.Size && fInfo.ModTime().UnixNano() == dbData.MDate {
		return false, nil
	}

	hash, err := internal.HashSumFile(fp)
	if err != nil {
		return false, err
	}
	fmt.Println("wrong")
	if res, err := r.stmtUpdFile.Exec(fInfo.Size(), fInfo.ModTime().UnixNano(), hash, dbData.Id); err != nil {
		return false, fmt.Errorf("stmtUpdFile error: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return false, fmt.Errorf("stmtUpdFile error: no rows affected in fact")
		}
		return true, nil
	}
}

//...
func (r *Repo) RemoveFile(id int64) error {
	if res, err := r.stmtDelFile.Exec(id); err != nil {
		return fmt.Errorf("stmtDelFile error: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return fmt.Errorf("stmtDelFile error: no rows added in fact")
		}
	}
	return nil
}

//...
func (r *Repo) CleanPacks() {
	//fmt.Println("очистка заблокированных пакетов из БД")
	for _, pack := range r.packages() { // проход по списку пакетов в БД
		if !r.IsActive(pack) {
			r.removePack(pack)
		}
	}

}

//...
func (r *Repo) packages() []string {
	var packs []string
	var name string
	rows, _ := r.db.Query("SELECT name FROM packages ORDER BY name;")
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&name)
		packs = append(packs, name)
	}
	return packs
}

//...
func (r *Repo) removePack(pack string) {
	res, err := r.db.Exec("DELETE FROM packages WHERE name=?;", pack)
	if err != nil {
		fmt.Println("error remove pack", pack)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		fmt.Println("должна быть удалена 1 запись: 0")
	}
	fmt.Println("удалено:", pack)
}

//...
func (r *Repo) SetPrepare() (err error) {
	//
	r.stmtAddFile, err = r.db.Prepare("INSERT INTO files ('package_id', 'path', 'size', 'mdate', 'hash')" +
		" VALUES (?, ?, ?, ?, ?);")
	if err != nil {
		return err
	}
	//
	r.stmtDelFile, err = r.db.Prepare("DELETE FROM files WHERE id=?;")
	if err != nil {
		return err
	}
	//
	r.stmtUpdFile, err = r.db.Prepare("UPDATE files SET size=?, mdate=?, hash=? WHERE id=?;")
	if err != nil {
		return err
	}
	return nil
}

// InitDB инициализирует файл db
func InitDB(path string) error {
	fp := dbPath(path)
	if internal.FileExists(fp) {
		fmt.Println("файл БД существует")
		return nil
	}
	db, err := newConnection(fp)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	if _, err := db.Exec(initSQL); err != nil {
		return err
	}
	fmt.Println("репозиторий проинициализирован")
	return nil
}

// newConnection возвращает соединение с БД или ошибку
func newConnection(fp string) (*sql.DB, error) {
	return sql.Open("sqlite3", fp)
}

//...
func dbPath(repoPath string) string {
	return filepath.Join(repoPath, fileDBName)
}
