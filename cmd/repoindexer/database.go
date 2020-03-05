package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const fileDBName string = "index.db"

// RepoObject объект репозитория с БД
type RepoObject struct {
	repoPath string
	db       *sql.DB
	fullMode bool
}

// NewRepoObj возвращает объект repoObj
func NewRepoObj(repoPath string) *RepoObject {
	repo := new(RepoObject)
	repo.repoPath = repoPath
	return repo
}

// SetFullMode устанавливает режим полной индексации
func (r *RepoObject) SetFullMode() {
	r.fullMode = true
	fmt.Println("установлен режим полной индексации")
}

// OpenDB открывает подключение к БД
func (r *RepoObject) OpenDB() error {
	fp := dbPath(r.repoPath)
	if !fileExists(fp) {
		return errors.New("репозиторий не инициализирован")
	}
	db, err := NewConnection(fp)
	if err != nil {
		return err
	}
	r.db = db
	return nil
}

// CloseDB закрывает DB соединение
func (r *RepoObject) CloseDB() {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			log.Println("error database close:", err)
		}
	}
}

// BlockedPacks возвращает список заблокированных для индексации пакетов в репозитории
func (r *RepoObject) BlockedPacks() []string {
	return []string{"blocked packet"}
}

// ActivePacks возвращает список пакетов в репозитории, за исключением заблокированных
func (r *RepoObject) ActivePacks() []string {
	return []string{"pack1", "pack2"}
}

// initDB инициализирует файл DB
func initDB(repoPath string) error {
	fp := dbPath(repoPath)
	if fileExists(fp) {
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
