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

const fileDBName string = "index.DB"

// Repo объект репозитория с БД
type Repo struct {
	RepoPath string
	DB       *sql.DB
	FullMode bool//?
	disabledPacks []string
}

// Структура с данными о файле пакета в БД
type FileInfo struct {
	Id int
	Path  string // путь файла относительно корневой папки пакета
	Size  int    // размер файла
	MDate time.Time    // дата изменения
	Hash  []byte // контрольная сумма
}

// NewRepoObj возвращает объект repoObj
func NewRepoObj(repoPath string) *Repo {
	repo := new(Repo)
	repo.RepoPath = repoPath
	return repo
}

// SetFullMode устанавливает режим полной индексации
func (r *Repo) SetFullMode() {
	r.FullMode = true
	fmt.Println("установлен режим полной индексации")
}

// OpenDB открывает подключение к БД
func (r *Repo) OpenDB() error {
	fp := dbPath(r.RepoPath)
	if !internal.FileExists(fp) {
		return errors.New("репозиторий не инициализирован")
	}
	db, err := NewConnection(fp)
	if err != nil {
		return err
	}
	r.DB = db
	return nil
}

// CloseDB закрывает DB соединение
func (r *Repo) CloseDB() {
	if r.DB != nil {
		if err := r.DB.Close(); err != nil {
			log.Println("error obj close:", err)
		}
	}
}

// ActivePacks возвращает список пакетов в репозитории, за исключением заблокированных
func (r *Repo) ActivePacks() []string {
	//todo add read pack from disk in repo
	fl := []string{
		//"aa_qwe",
		"a_pack",
		"b_pack",
		"e_pack",
	}
	activeList := make([]string, len(fl))
	for _, name := range fl {
		if r.packIsBlocked(name) {
			continue
		} else {
			activeList = append(activeList, name)
		}
	}
	return activeList
}

// FilesPackRepo возвращает список файлов указанного пакета в репозитории
func (r *Repo) FilesPackRepo(pack string) ([]string, error) {
	path := filepath.Join(r.RepoPath, pack) // base Path repopath/packname
	fList := make([]string,0,50) // reserve place for ~50 files
	fpCh := make(chan string) // channel for filepath
	erCh := make(chan error) // channel for error
	unWanted, _ := regexp.Compile("(.*[Tt]humb\\.db)|(.*~.*)")

	//go walkPack(Path, fpCh, erCh)
	go func (root string, fpCh chan<- string, erCh chan<- error) {
		err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
			if er != nil {
				// debug message: er
				return fmt.Errorf("не найден пакет: %q\n", fp)
			}
			if info.IsDir() { // skip directory
				return nil
			} else if unWanted.MatchString(fp){ // skip unwanted file
				//fmt.Println("skip unwanted:", fp) // todo add to log.debug
				return nil
			}
			fp, _ = filepath.Rel(path, fp) // trim base Path repopath/packname
			fpCh<- fp
			return nil
			})
		if err != nil {
			erCh<- err
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
	err := r.DB.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := r.DB.Exec("INSERT INTO packages VALUES (null, ?, '', null)", pack)
		if err != nil {
			log.Fatalf("create pack [ %v ] record in DB: %v", pack, err)
		}
		id, _ = res.LastInsertId()
	}

	cnt := 0
	if err := r.DB.QueryRow("SELECT COUNT(*) FROM FILES WHERE package_id=?;", id).Scan(&cnt); err != nil {
		return nil, err
	}
	if cnt == 0 {
		return []FileInfo{}, nil
	}

	rows, err := r.DB.Query("SELECT id, path, size, mdate, hash FROM files WHERE package_id=?", id)
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

// PrepareDisabledPacksList устанавливает список заблокированных пакетов репозитория
func (r *Repo) PrepareDisabledPaksList() {
	rows, err := r.DB.Query("SELECT name FROM excludes;")
	if err != nil {
		log.Fatalf("error select disabled packs: %v", err)
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		rows.Scan(&name)
		r.disabledPacks = append(r.disabledPacks, name)
	}
}

func (r *Repo) packIsBlocked(name string) bool {
	for i := range r.disabledPacks {
		fmt.Println("dsl check:", name)
		if name == r.disabledPacks[i] {
			return true
		}
	}
	return false
}

// InitDB инициализирует файл DB
func InitDB(repoPath string) error {
	fp := dbPath(repoPath)
	if internal.FileExists(fp) {
		fmt.Println("файл БД существует")
		return nil
	}
	db, err := NewConnection(fp)
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

// NewConnection возвращает соединение с БД или ошибку
func NewConnection(fp string) (*sql.DB, error) {
	return sql.Open("sqlite3", fp)
}

//...
func dbPath(repoPath string) string {
	return filepath.Join(repoPath, fileDBName)
}
