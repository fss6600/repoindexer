package obj

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/proc"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const fileDBName string = "index.DB"

// Repo объект репозитория с БД
type Repo struct {
	RepoPath string
	DB       *sql.DB
	FullMode bool
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
	if !proc.FileExists(fp) {
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

// BlockedPacks возвращает список заблокированных для индексации пакетов в репозитории
func (r *Repo) BlockedPacks() []string {
	return []string{"blocked packet"}
}

// ActivePacks возвращает список пакетов в репозитории, за исключением заблокированных
func (r *Repo) ActivePacks() []string {
	return []string{"pack1", "pack2"}
}

// InitDB инициализирует файл DB
func InitDB(repoPath string) error {
	fp := dbPath(repoPath)
	if proc.FileExists(fp) {
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



// functions
// NewConnection возвращает соединение с БД или ошибку
func NewConnection(fp string) (*sql.DB, error) {
	return sql.Open("sqlite3", fp)
}

//...
func dbPath(repoPath string) string {
	return filepath.Join(repoPath, fileDBName)
}
