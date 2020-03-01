package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const fileDBName string = "index.db"

//...
type dbObj string

// RepoObject объект репозитория с БД
type RepoObject struct {
	repoPath string
	db       dbObj
	fullMode bool
}

// NewRepoObj возвращает объект repoObj
func NewRepoObj(repoPath string) (*RepoObject, error) {
	fileDB := filepath.Join(repoPath, fileDBName)
	if _, err := os.Stat(fileDB); err != nil {
		return nil, errors.New("БД не инициализирована")
	}
	repo := new(RepoObject)
	repo.repoPath = repoPath
	return repo, nil
}

// SetFullMode устанавливает режим полной индексации
func (r *RepoObject) SetFullMode() {
	r.fullMode = true
	fmt.Println("установлен режим полной индексации")
}

// Close закрывает DB
func (r *RepoObject) Close() {
	fmt.Println("DB closed")
}
