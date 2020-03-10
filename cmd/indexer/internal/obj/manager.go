package obj

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const fileDBName string = "index.db"

// Repo объект репозитория с БД
type Repo struct {
	path string
	db   *sql.DB
	//FullMode      bool //?
	disPacks []string
	actPacks []string
}

// Структура с данными о файле пакета в БД
type FileInfo struct {
	Id    int
	Path  string    // путь файла относительно корневой папки пакета
	Size  int       // размер файла
	MDate time.Time // дата изменения
	Hash  []byte    // контрольная сумма
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
	r.db = db
	return nil
}

// Close закрывает db соединение
func (r *Repo) Close() {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			log.Println("error close db:", err)
		}
	}
}

// ActivePacks кэширует и возвращает список пакетов в репозитории, за исключением заблокированных
func (r *Repo) ActivePacks() []string {
	if len(r.actPacks) == 0 {

		//todo add read pack from disk in repo
		fl := []string{
			//"aa_qwe",
			"a_pack",
			"b_pack",
			"e_pack",
		}

		for _, name := range fl {
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

// FilesPackDB возвращвет список файлов указанного пакета имеющихся в БД
func (r *Repo) FilesPackDB(pack string) ([]FileInfo, error) {
	var id int64
	err := r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := r.db.Exec("INSERT INTO packages VALUES (null, ?, '', null);", pack)
		if err != nil {
			log.Fatalf("create pack [ %v ] record in db: %v", pack, err)
		}
		id, _ = res.LastInsertId()
	}

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

	fdataList := make([]FileInfo, cnt)
	for rows.Next() {
		fd := new(FileInfo)
		if err := rows.Scan(&fd.Id, &fd.Path, &fd.Size, &fd.MDate, &fd.Hash); err != nil {
			return nil, err
		}
		fdataList = append(fdataList, *fd)
	}
	return fdataList, nil
}

func (r *Repo) packIsBlocked(name string) bool {
	for _, fn := range r.DisabledPacks() {
		fmt.Println("dsl check:", name)
		if name == fn {
			return true
		}
	}
	return false
}

//...
func (r *Repo) CheckExists(fn string) error {
	for _, fp := range r.ActivePacks() {
		if fp == fn {
			return nil
		}
	}
	return fmt.Errorf("пакет [ %v ] не найден в репозитории или заблокирован", fn)
}

func (r *Repo) Clean() {
	fmt.Println("здесь будет очистка заблокированных пакетов из БД")
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
