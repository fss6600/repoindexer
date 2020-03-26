package obj

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mattn/go-sqlite3"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const (
	fileDBName     string = "index.db"
	Indexgz        string = "index.gz"
	DBVersionMajor int64  = 1
	DBVersionMinor int64  = 3
)

var err error

type ErrAlias error

// Repo объект репозитория с БД
type Repo struct {
	path        string
	db          *sql.DB
	disPacks    []string  // список заблокированных пакетов
	actPacks    []string  // список активных (актуальных) пакетов
	indPacks    []string  // список проиндексированных пакетов
	stmtAddFile *sql.Stmt // предустановка запроса на добавление данных файла пакета в БД
	stmtDelFile *sql.Stmt // предустановка запроса на удаление данных файла пакетав БД
	stmtUpdFile *sql.Stmt // предустановка запроса на изменение данных файла пакета в БД
}

// FileInfo структура с данными о файле пакета в БД
type FileInfo struct {
	Id    int64
	Path  string // путь файла относительно корневой папки пакета
	Size  int64  // размер файла
	MDate int64  // дата изменения
	Hash  string // контрольная сумма
}

// NewRepoObj возвращает объект repoObj
func NewRepoObj(path string) (*Repo, error) {
	repo := new(Repo)
	if err = repo.SetPath(path); err != nil {
		return nil, err
	}
	return repo, nil
}

//...
func (r *Repo) SetPath(path string) error {
	if path == "" {
		return fmt.Errorf("не указан путь к репозиторию")
	}
	r.path = path
	return nil
}

//...
func (r *Repo) Path() string {
	return r.path
}

// OpenDB открывает подключение к БД
func (r *Repo) OpenDB() error {
	fp := dbPath(r.path)
	if !utils.FileExists(fp) {
		return fmt.Errorf("Репозиторий не инициализирован")
	}
	db, err := newConnection(fp)
	if err != nil {
		return err
	}
	r.db = db
	if err = r.checkDB(); err != nil {
		return err
	}
	// инициализация поддержки первичных ключей в БД
	if _, err := r.db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("init PRAGMA failed", err)
	}
	return nil
}

// Close закрывает db соединение
func (r *Repo) Close() error {
	if r.stmtAddFile != nil {
		_ = r.stmtAddFile.Close()
	}
	if r.stmtDelFile != nil {
		_ = r.stmtDelFile.Close()
	}
	if r.stmtUpdFile != nil {
		_ = r.stmtUpdFile.Close()
	}
	// упаковка данных, переиндексация
	if r.db != nil {
		if _, err := r.db.Exec("PRAGMA optimize;"); err != nil {
			return fmt.Errorf("ошибка оптимизации БД: ", err)
		}
		if err = r.db.Close(); err != nil {
			return fmt.Errorf("ошибка закрытия БД:", err)
		}
	}
	return nil
}

func (r *Repo) Clean() error {
	if r.db != nil {
		if _, err = r.db.Exec("VACUUM"); err != nil {
			return fmt.Errorf("ошибка очистки БД: ", err)
		}
		if _, err = r.db.Exec("REINDEX"); err != nil {
			return fmt.Errorf("ошибка перестройки идекса таблиц БД: ", err)
		}
	}
	return nil
}

// PackageID возвращает ID пакета
func (r *Repo) PackageID(pack string) (int64, error) {
	var id int64
	if err = r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id); err == sql.ErrNoRows {
		// пакета нет в БД - добавляем
		res, err := r.db.Exec("INSERT INTO packages ('name', 'hash') VALUES (?, 0);", pack)
		if err != nil {
			return 0, fmt.Errorf("create pack [ %v ] record in db: %v", pack, err)
		}
		id, _ = res.LastInsertId()
	}
	return id, nil
}

// Alias возвращает псевдоним пакета при наличии
func (r *Repo) Alias(pack string) (alias string) {
	_ = r.db.QueryRow("SELECT alias FROM aliases WHERE Name=?;", pack).Scan(&alias)
	return
}

// Alias возвращает срез срезов (пар) псевдоним-пакет
func (r *Repo) Aliases() [][]string {
	var aliases [][]string
	var alias, name string
	rows, _ := r.db.Query("SELECT alias, Name FROM aliases ORDER BY alias;")
	defer rows.Close()
	for rows.Next() {
		var aliasPair []string
		_ = rows.Scan(&alias, &name)
		aliasPair = append(aliasPair, name, alias)
		aliases = append(aliases, aliasPair)
	}
	return aliases
}

// SetAlias устанавливает псевдоним для пакета при отсутствии уже установленного псевдонима
// и при наличии актуального пакета
func (r *Repo) SetAlias(alias []string) error {
	if !r.PackIsActive(alias[0]) {
		return ErrAlias(fmt.Errorf("пакет [ %v ] не найден или заблокирован\n", alias[0]))
	}
	if res, err := r.db.Exec("INSERT INTO aliases (name, alias) VALUES (?, ?);", alias[0], alias[1]); err != nil {
		switch err.(type) {
		case sqlite3.Error:
			if err.(sqlite3.Error).Code == sqlite3.ErrConstraint {
				return ErrAlias(fmt.Errorf("Псевдоним [ %v ] или псевдоним для пакета [ %v ] уже заданы", alias[1], alias[0]))
			}
		default:
			return fmt.Errorf(":manager: %v", err)
		}
	} else if c, err := res.RowsAffected(); err != nil || c != 1 {
		return fmt.Errorf(":manager: псевдоним не добавлен: err=%v;count=%d", err, c)
	}
	return nil
}

// DelAlias удалает псевдоним
func (r *Repo) DelAlias(alias string) error {
	if res, err := r.db.Exec("DELETE FROM aliases WHERE alias=?;", alias); err != nil {
		return fmt.Errorf(":manager: %v", err)
	} else if c, _ := res.RowsAffected(); c != 1 {
		return ErrAlias(fmt.Errorf("не найден псевдоним [ %v ]", alias))
	}
	return nil
}

//... IndexedPacks
func (r *Repo) IndexedPacks() []string { //todo в горутину
	if len(r.indPacks) == 0 {
		ch := make(chan string)
		go r.indexedPack(ch)
		for name := range ch {
			r.indPacks = append(r.indPacks, name)
		}
	}
	return r.indPacks
}

//... NoIndexedPacks
func (r *Repo) NoIndexedPacks() []string { //todo в горутину
	lst := make([]string, 0)
	for _, pack := range r.ActivePacks() {
		if !r.PackIsIndexed(pack) {
			lst = append(lst, pack)
		}
	}
	return lst
}

// ActivePacks кэширует и возвращает список пакетов в репозитории, за исключением заблокированных
func (r *Repo) ActivePacks() []string { //todo в горутину
	if len(r.actPacks) == 0 {
		ch := make(chan string, 3)
		go utils.DirList(r.Path(), ch)
		for name := range ch {
			if r.PackIsBlocked(name) {
				continue
			} else {
				r.actPacks = append(r.actPacks, name)
			}
		}
	}
	return r.actPacks
}

// DisabledPacks кэширует и возвращает список заблокированных пакетов репозитория
func (r *Repo) DisabledPacks() []string { //todo в горутину
	if len(r.disPacks) == 0 {
		rows, err := r.db.Query("SELECT Name FROM excludes;")
		if err != nil {
			panic(fmt.Errorf("error select disabled packs: %v", err))
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

type HashedPackData struct {
	Id    int64             `json:"-"`
	Name  string            `json:"-"`
	Alias string            `json:"alias"`
	Hash  string            `json:"phash"`
	Files map[string]string `json:"files"`
}

//...
func (r *Repo) HashedPackages(packs chan HashedPackData) error {
	rows, err := r.db.Query("SELECT id, Name, hash FROM packages ORDER BY Name;")
	if err == sql.ErrNoRows {
		close(packs)
		return nil
	} else if err != nil {
		return fmt.Errorf("HashedPackages: %v", err)
	}
	defer rows.Close()

	var pData HashedPackData
	for rows.Next() {
		if err = rows.Scan(&pData.Id, &pData.Name, &pData.Hash); err != nil {
			return fmt.Errorf("HashedPackages: %v", err)
		}
		pData.Alias = r.Alias(pData.Name)

		f := map[string]string{}
		fList, _ := r.FilesPackDB(pData.Id)

		for _, fd := range fList {
			f[fd.Path] = fd.Hash
		}
		pData.Files = f

		packs <- pData
	}
	close(packs)
	return nil
}

// FilesPackRepo возвращает список файлов указанного пакета в репозитории // todo - на горутины с передачей данных через канал
func (r *Repo) FilesPackRepo(pack string) ([]string, error) {
	path := filepath.Join(r.path, pack) // base Path repopath/packname
	fList := make([]string, 0, 50)      // reserve place for ~50 files
	fpCh := make(chan string)           // channel for filepath
	erCh := make(chan error)            // channel for error
	unWanted, _ := regexp.Compile("(.*[Tt]humb[s]?\\.db)|(.*~.*)")

	// --- go walkPack(Path, fpCh, erCh)
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
	// ---

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
func (r *Repo) FilesPackDB(id int64) ([]FileInfo, error) { // todo - на горутины с передачей данных через канал
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

func (r *Repo) PackIsIndexed(name string) bool {
	for _, fn := range r.IndexedPacks() {
		if name == fn {
			return true
		}
	}
	return false
}

//...
func (r *Repo) PackIsBlocked(name string) bool {
	for _, fn := range r.DisabledPacks() {
		if name == fn {
			return true
		}
	}
	return false
}

//...
func (r *Repo) PackIsActive(pack string) bool {
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
	hash, err := utils.HashSumFile(fp)
	if err != nil {
		return err
	}
	if res, err := r.stmtAddFile.Exec(id, fPath, fInfo.Size(), fInfo.ModTime().UnixNano(), hash); err != nil {
		return fmt.Errorf(":stmtAddFile: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return fmt.Errorf(":stmtAddFile: no rows added in fact")
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

	hash, err := utils.HashSumFile(fp)
	if err != nil {
		return false, err
	}
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
		return fmt.Errorf(":stmtDelFile: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return fmt.Errorf(":stmtDelFile: no rows added in fact")
		}
	}
	return nil
}

//...
func (r *Repo) CleanPacks() error {
	// clean cached lists
	r.actPacks = []string{}
	r.disPacks = []string{}
	r.indPacks = []string{}
	for _, pack := range r.packages() { // проход по списку пакетов в БД
		if !r.PackIsActive(pack) {
			if err = r.RemovePack(pack); err != nil {
				return err
			}
		}
	}
	return nil
}

// indexedPack передает через канал список проиндексированных пакетов репозитория
func (r *Repo) indexedPack(ch chan<- string) {
	var pack string
	rows, _ := r.db.Query("SELECT name FROM packages ORDER BY name;")
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&pack)
		ch <- pack
	}
	close(ch)
}

//...
func (r *Repo) packages() []string { // todo: в горутину ^^^
	var packs []string
	var name string
	rows, _ := r.db.Query("SELECT Name FROM packages ORDER BY Name;")
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&name)
		packs = append(packs, name)
	}
	return packs
}

//...
func (r *Repo) DisablePack(pack string) error {
	res, err := r.db.Exec("INSERT INTO excludes VALUES (?);", pack)
	if err != nil {
		return fmt.Errorf(":DisablePack: %v", pack)
	}
	if c, _ := res.RowsAffected(); c != 1 {
		return fmt.Errorf(":DisablePack:[ %v ]:добавлено %d, должно 1", pack, c)
	}
	fmt.Printf("заблокирован: [ %s ]\n", pack)
	return nil
}

//...
func (r *Repo) EnablePack(pack string) error {
	res, err := r.db.Exec("DELETE FROM excludes WHERE Name=?;", pack)
	if err != nil {
		return fmt.Errorf(":EnablePack: %v", pack)
	}
	if c, _ := res.RowsAffected(); c != 1 {
		return fmt.Errorf(":EnablePack:[ %v ]:удалено %d, должно 1", pack, c)
	}
	fmt.Printf("активирован: [ %s ]\n", pack)
	return nil
}

//...
func (r *Repo) RemovePack(pack string) error {
	res, err := r.db.Exec("DELETE FROM packages WHERE Name=?;", pack)
	if err != nil {
		return fmt.Errorf("error remove pack", pack)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("должна быть удалена 1 запись: 0")
	}
	fmt.Printf("  - [ %s ]\n", pack)
	return nil
}

func (r *Repo) DBCleanPackages() error {
	_, err := r.db.Exec("DELETE FROM packages;")
	if err != nil {
		return fmt.Errorf(":DBCleanPackages:%v", err)
	}
	return nil
}

func (r *Repo) DBCleanAliases() error {
	_, err := r.db.Exec("DELETE FROM aliases;")
	if err != nil {
		return fmt.Errorf(":DBCleanAliases:%v", err)
	}
	return nil
}

func (r *Repo) DBCleanStatus() error {
	_, err := r.db.Exec("DELETE FROM excludes;")
	if err != nil {
		return fmt.Errorf(":DBCleanStatus:%v", err)
	}
	return nil
}

// todo: пересмотреть на расчет суммы частями
//...
func (r *Repo) HashSumPack(id int64) error {
	var hash, hTotal string
	rows, _ := r.db.Query("SELECT hash FROM FILES WHERE package_id=?;", id)
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&hash)
		hTotal += hash
	}
	hash = utils.HashSum(hTotal)
	res, err := r.db.Exec("UPDATE packages SET hash=? WHERE id=?;", hash, id)
	if err != nil {
		return fmt.Errorf("HashSumPack: %v", err)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("HashSumPack: должна быть обновлена 1 запись: 0")
	}
	return nil
}

//...
func (r *Repo) SetPrepare() error {
	//
	sqlExpr := "INSERT INTO files ('package_id', 'path', 'size', 'mdate', 'hash') VALUES (?, ?, ?, ?, ?);"
	r.stmtAddFile, err = r.db.Prepare(sqlExpr)
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

// ...
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

//...
func (r *Repo) Status() (*RepoStData, error) {
	data := new(RepoStData)
	// количество активных пакетов
	if err = r.db.QueryRow("SELECT COUNT() FROM packages;").Scan(&data.IndexedCnt); err != nil {
		return nil, fmt.Errorf("manager::Status::Active: %v", err)
	}

	// количество заблокированных
	if err = r.db.QueryRow("SELECT COUNT() FROM excludes;").Scan(&data.BlockedCnt); err != nil {
		return nil, fmt.Errorf("manager::Status::Blocked: %v", err)
	}

	// количество пакетов в репозитории
	ch := make(chan string)
	go utils.DirList(r.Path(), ch)
	for _ = range ch {
		data.TotalCnt += 1
	}

	// данные БД файла
	fInfo, err := os.Stat(dbPath(r.Path()))
	if err != nil {
		data.DBSize = -1
		data.DBMDate = time.Time{}
	} else {
		data.DBSize = fInfo.Size()
		data.DBMDate = fInfo.ModTime()
	}

	// данные индекс файла
	fInfo, err = os.Stat(filepath.Join(r.Path(), Indexgz))
	if err != nil {
		data.IndexSize = -1
		data.IndexMDate = time.Time{}
	} else {
		data.IndexSize = fInfo.Size()
		data.IndexMDate = fInfo.ModTime()
	}

	// данные индекс-хэш файла
	fInfo, err = os.Stat(filepath.Join(r.Path(), Indexgz+".sha1"))
	if err != nil {
		data.HashSize = -1
		data.HashMDate = time.Time{}
	} else {
		data.HashSize = fInfo.Size()
		data.HashMDate = fInfo.ModTime()
	}
	return data, nil
}

type ListData struct {
	Status int8
	Name   string
}

// ...
func (r *Repo) List(ch chan<- *ListData) {
	dir := make(chan string)
	go utils.DirList(r.Path(), dir)
	for name := range dir {
		data := new(ListData)
		alias := r.Alias(name)
		if alias != "" {
			data.Name = fmt.Sprintf("%v (%v)", name, alias)
		} else {
			data.Name = name
		}
		if r.PackIsBlocked(name) {
			// блок
			data.Status = 0
		} else if r.PackIsIndexed(name) {
			// актл
			data.Status = 1
		} else {
			// !инд
			data.Status = -1
		}
		ch <- data
	}
	close(ch)
}

func (r *Repo) checkDB() error {
	if rows, err := r.db.Query("PRAGMA integrity_check ;"); err != nil {
		return err
	} else {
		var res interface{}
		for rows.Next() {
			err = rows.Scan(&res)
			if res != "ok" || err != nil {
				return fmt.Errorf("ошибка целостности БД; требуется повторная инициализация %v", err)
			}
		}
	}

	vmaj, vmin, err := r.VersionDB()
	if err != nil {
		return err
	} // todo - миграцию при поднятии версии программы
	if vmaj < DBVersionMajor || vmin < DBVersionMinor {
		return fmt.Errorf("версия БД [%d.%d] не соответствует требуемой[%d.%d]; произведите инициализацию", // миграцию?
			vmaj, vmin, DBVersionMajor, DBVersionMinor)
	} else if vmaj > DBVersionMajor || vmin > DBVersionMinor {
		return fmt.Errorf("версия БД [%d.%d] старше требуемой[%d.%d]; возможно вы используете старую версию программы",
			vmaj, vmin, DBVersionMajor, DBVersionMinor)
	}
	return nil
}

//..
func (r *Repo) VersionDB() (int64, int64, error) {
	var vmaj, vmin int64
	err := r.db.QueryRow("SELECT vers_major, vers_minor FROM info WHERE id=1;").Scan(&vmaj, &vmin)
	if err != nil {
		return 0, 0, fmt.Errorf("нет данных о версии БД; произведите инициализацию")
	}
	return vmaj, vmin, nil
}

// InitDB инициализирует файл db
func InitDB(path string) error {
	fp := dbPath(path)
	if utils.FileExists(fp) {
		return fmt.Errorf("попытка повторной инициализации")
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
	if _, err := db.Exec("INSERT INTO info (id, vers_major, vers_minor) VALUES (?, ?, ?);",
		1, DBVersionMajor, DBVersionMinor); err != nil {
		return err
	}
	fmt.Println("Репозиторий инициализирован")
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
