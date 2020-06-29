package obj

import "time"

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
	IndexMDate time.Time // дата изменения индекс-файла
	DBSize     int64     // размер файла БД в байтах
	DBMDate    time.Time // дата изменения файла БД
	HashSize   int64     // размер хэш-файла в байтах
	HashMDate  time.Time // дата изменения хэш-файла
}

// ListData структура для сбора и передачи данных списка пакетов
type ListData struct {
	Status int8
	Name   string
}
