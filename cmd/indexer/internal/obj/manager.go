package obj

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	//"regexp"
	"time"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const (
	fileDBName     string = "index.db"
	Indexgz        string = "index.gz"
	DBVersionMajor int64  = 1
	DBVersionMinor int64  = 4
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
	ID    int64  // package ID
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

// SetPath ...
func (r *Repo) SetPath(path string) error {
	if path == "" {
		return fmt.Errorf("не указан путь к репозиторию")
	}
	r.path = path
	return nil
}

// Path ...
func (r *Repo) Path() string {
	return r.path
}

// OpenDB открывает подключение к БД
func (r *Repo) OpenDB() error {
	fp := DBPath(r.path)
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
		r.db = nil
	}
	return nil
}

// Clean ...
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

// GetPackageID возвращает ID пакета
func (r *Repo) GetPackageID(pack string) (int64, error) {
	var id int64
	if err = r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id); err == sql.ErrNoRows {
		return 0, err
	} else if err != nil {
		panic(fmt.Errorf("ошибка получения ID пакета [ %v ]: %v", pack, err))
	}
	return id, nil
}

// NewPackage ...
func (r *Repo) NewPackage(name string) (int64, error) {
	var id int64
	res, err := r.db.Exec("INSERT INTO packages ('name', 'hash') VALUES (?, 0);", name)
	if err != nil {
		return 0, fmt.Errorf("create pack [ %v ] record in db: %v", name, err)
	}
	id, _ = res.LastInsertId()
	return id, nil
}

// Alias возвращает псевдоним пакета при наличии
func (r *Repo) Alias(pack string) (alias string) {
	_ = r.db.QueryRow("SELECT alias FROM aliases WHERE Name=?;", pack).Scan(&alias)
	return
}

// Aliases возвращает срез срезов (пар) псевдоним-пакет
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
	fmt.Printf("Установлен псевдоним: [ %v ]=( %v )\n", alias[0], alias[1])
	return nil
}

// DelAlias удалает псевдоним
func (r *Repo) DelAlias(alias string) error {
	if res, err := r.db.Exec("DELETE FROM aliases WHERE alias=?;", alias); err != nil {
		return fmt.Errorf(":manager: %v", err)
	} else if c, _ := res.RowsAffected(); c != 1 {
		return ErrAlias(fmt.Errorf("не найден псевдоним [ %v ]", alias))
	}
	fmt.Printf("Удален псевдоним: [ %v ]\n", alias)
	return nil
}

// IndexedPacks ...
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

// NoIndexedPacks ...
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
		rows, err := r.db.Query("SELECT name FROM excludes;")
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

// HashedPackData ...
type HashedPackData struct {
	ID    int64             `json:"-"`
	Name  string            `json:"-"`
	Alias string            `json:"alias"`
	Hash  string            `json:"phash"`
	Exec  string            `json:"execf"`
	Files map[string]string `json:"files"`
}

// HashedPackages ...
func (r *Repo) HashedPackages(packs chan HashedPackData) error {
	defer close(packs)
	rows, err := r.db.Query("SELECT id, name, hash, exec FROM packages ORDER BY name;")
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return fmt.Errorf("HashedPackages: %v", err)
	}
	defer rows.Close()

	var pData HashedPackData
	var filesPackDB []*FileInfo

	for rows.Next() {
		if err = rows.Scan(&pData.ID, &pData.Name, &pData.Hash, &pData.Exec); err != nil {
			return fmt.Errorf("HashedPackages: %v", err)
		}
		pData.Alias = r.Alias(pData.Name)

		if filesPackDB, err = r.FilesPackDB(pData.ID); err != nil {
			return err
		}
		files := map[string]string{}

		for _, fd := range filesPackDB {
			files[fd.Path] = fd.Hash
		}
		pData.Files = files
		packs <- pData
	}
	return nil
}

// FilesPackRepo возвращает список файлов указанного пакета в репозитории
func (r *Repo) FilesPackRepo(pack string) ([]*FileInfo, error) {
	path := filepath.Join(r.path, pack) // base Path repopath/packname
	fList := make([]*FileInfo, 0, 50)   // reserve place for ~50 files
	fInfoCh := make(chan *FileInfo)     // channel for filepath
	erCh := make(chan error)            // channel for error
	//unWanted, _ := regexp.Compile("(.*[Tt]humb[s]?\\.db)|(.*~.*)")

	go DirWalk(path, fInfoCh, erCh)

	for {
		select {
		case err := <-erCh:
			return nil, err
		case fInfo, more := <-fInfoCh:
			//if ok && !unWanted.MatchString(fp) {
			if more {
				fList = append(fList, fInfo)
			} else {
				return fList, nil
			}
		}
	}
}

// FilesPackDBCount возвращает количество проиндексированных файлов в пакете
// func (r *Repo) FilesPackDBCount(packID int64) (int64, error) {
// 	var cnt int64
// 	err := r.db.QueryRow("SELECT COUNT(*) FROM FILES WHERE package_id=?;", packID).Scan(&cnt)
// 	return cnt, err
// }

// FilesPackDB возвращвет список файлов пакета имеющихся в БД
func (r *Repo) FilesPackDB(id int64) ([]*FileInfo, error) {
	rows, err := r.db.Query("SELECT id, path, size, mdate, hash FROM files WHERE package_id=? ORDER BY path;", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pFileInfoList []*FileInfo

	for rows.Next() {
		fd := new(FileInfo)
		if err := rows.Scan(&fd.ID, &fd.Path, &fd.Size, &fd.MDate, &fd.Hash); err != nil {
			return nil, err
		}
		pFileInfoList = append(pFileInfoList, fd)
	}
	return pFileInfoList, nil
}

// PackIsIndexed ...
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
func (r *Repo) AddFile(fInfo *FileInfo) error {
	if res, err := r.stmtAddFile.Exec(fInfo.ID, fInfo.Path, fInfo.Size, fInfo.MDate, fInfo.Hash); err != nil {
		return fmt.Errorf(":stmtAddFile: %v", err)
	} else {
		if ret, _ := res.RowsAffected(); ret == 0 {
			return fmt.Errorf(":stmtAddFile: no rows added in fact")
		}
	}
	return nil
}

//..
func (r *Repo) UpdateFileData(fd *FileInfo) error {
	if res, err := r.stmtUpdFile.Exec(fd.Size, fd.MDate, fd.Hash, fd.ID); err != nil {
		return fmt.Errorf("stmtUpdFile error: %v", err)
	} else if ret, _ := res.RowsAffected(); ret == 0 {
		return fmt.Errorf("stmtUpdFile error: no rows affected in fact")
	}
	return nil
}

//...
func (r *Repo) RemoveFile(fInfo *FileInfo) error {
	if res, err := r.stmtDelFile.Exec(fInfo.ID); err != nil {
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
	fInfo, err := os.Stat(DBPath(r.Path()))
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
	return nil
}

func (r *Repo) CheckDBVersion() error {
	vmaj, vmin, err := r.VersionDB()
	if err != nil {
		return err
	} // todo - миграцию при поднятии версии программы
	if DBVersionMajor > vmaj {
		return fmt.Errorf("\n\tУстаревшая версия репозитория. Требуется переиндексация")
	} else if DBVersionMinor > vmin {
		return fmt.Errorf("\n\tТребуется миграция БД репозитория")
	} else if vmaj > DBVersionMajor || vmin > DBVersionMinor {
		return fmt.Errorf("\n\tверсия БД [%d.%d] старше требуемой[%d.%d]; возможно вы используете старую версию программы",
			vmaj, vmin, DBVersionMajor, DBVersionMinor)
	}
	return nil
}

func (r *Repo) EmptyExecFilesList() []string {
	var name string
	var emptyList []string

	rows, _ := r.db.Query("SELECT name FROM packages WHERE exec is null;")
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&name)
		emptyList = append(emptyList, name)
	}
	return emptyList
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

func (r *Repo) ExecFileSet(pack string, force bool) error {
	id, err := r.GetPackageID(pack)
	utils.CheckError(fmt.Sprintf("не найден пакет в БД: %v", pack), &err)

	switch force {
	case false:
		var exec_db string
		if err := r.db.QueryRow(
			"SELECT CASE WHEN exec IS null THEN '' ELSE exec END exec FROM packages WHERE id=?;", id).Scan(
			&exec_db); err != nil {
			//if err := r.db.QueryRow("SELECT exec FROM packages WHERE id=?;", id).Scan(&exec_db); err != nil {
			return err
		}
		if exec_db != "" {
			fmt.Printf("\t%v: уже установлен, пропуск\n", pack)
			return nil
		}
		fallthrough
	case true:
		execFile := defineExecFile(r, pack)
		res, err := r.db.Exec("UPDATE packages SET exec=? WHERE id=?;", execFile, id)
		if err != nil {
			return fmt.Errorf("ExecFileSet: %v", err)
		}
		if c, _ := res.RowsAffected(); c == 0 {
			return fmt.Errorf("ExecFileSet: должна быть обновлена 1 запись: 0")
		}
		fmt.Printf("\t%v: установлен в [ %v ]\n", pack, execFile)
	}
	return nil
}

func (r *Repo) ExecFileDel(pack string) error {
	id, err := r.GetPackageID(pack)
	utils.CheckError(fmt.Sprintf("не найден пакет в БД: %v", pack), &err)
	res, err := r.db.Exec("UPDATE packages SET exec='noexec' WHERE id=?;", id)
	if err != nil {
		return fmt.Errorf("ExecFileDel: %v", err)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("ExecFileDel: должна быть обновлена 1 запись: 0")
	}
	fmt.Printf("Исполняемый файл пакета '%v' установлен в 'noexec' \n", pack)
	return nil
}

func (r *Repo) ExecFileInfo(pack string) (string, error) {
	id, err := r.GetPackageID(pack)
	utils.CheckError(fmt.Sprintf("не найден пакет в БД: %v", pack), &err)
	var exec_db string

	if err := r.db.QueryRow(
		"SELECT CASE WHEN exec IS null THEN '' ELSE exec END exec FROM packages WHERE id=?;", id).Scan(
		&exec_db); err != nil {
		//if err := r.db.QueryRow("SELECT exec FROM packages WHERE id=?;", id).Scan(&exec_db); err != nil {
		return "", err
	}
	return exec_db, nil
}

func (r *Repo) CheckEmptyExecFiles() error {
	if len(r.EmptyExecFilesList()) > 0 {
		return fmt.Errorf("\n\tТребуется определить исполняемые файлы\n\t" +
			"Запустите программу с командой 'exec check'\n")
	}
	return nil
}

// InitDB инициализирует файл db
func InitDB(path string) error {
	fp := DBPath(path)
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
func DBPath(repoPath string) string {
	return filepath.Join(repoPath, fileDBName)
}

//...
func CleanForMigrate(repo *Repo) error {
	for _, fp := range []string{fileDBName, Indexgz, Indexgz + ".sha1"} {
		fp = filepath.Join(repo.Path(), fp)
		_ = os.Remove(fp)
	}
	return nil
}
