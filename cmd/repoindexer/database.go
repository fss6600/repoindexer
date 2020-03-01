package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const fileDBName string = "index.db"

//...
//type dbObj

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
	if !fileDbExists(r.repoPath) {
		return errors.New("БД не инициализирована")
	}

	db, err := getConnection(r.repoPath)
	if err != nil {
		return err
	}

	r.db = db
	return nil
}

// CloseDB закрывает DB соединение
func (r *RepoObject) CloseDB() {
	if err := r.db.Close(); err != nil {
		log.Printf("ERROR: database close - %s\n", err)
	}
}

// initDB инициализирует файл DB
func initDB(repoPath string) error {

	if fileDbExists(repoPath) {
		fmt.Println("файл БД существует")
		return nil
	}

	db, err := getConnection(repoPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = db.Close()
	}()

	if _, err := db.Exec("CREATE TABLE excludes(id INTEGER PRIMARY KEY AUTOINCREMENT, " +
		"name VARCHAR NOT NULL UNIQUE);"); err != nil {
		return err
	}
	fmt.Println("репозиторий проинициализирован")
	return nil
}

//...
func fileDbExists(repoPath string) bool {
	fp := filepath.Join(repoPath, fileDBName)
	_, err := os.Stat(fp)
	switch err {
	case nil:
		return true
	default:
		return false
	}
}

// getConnection возвращает соединение с БД или ошибку
func getConnection(repoPath string) (*sql.DB, error) {
	fp := filepath.Join(repoPath, fileDBName)
	return sql.Open("sqlite3", fp)
}
