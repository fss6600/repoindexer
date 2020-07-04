package handler

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"time"

	"github.com/mattn/go-sqlite3"
)

// NewRepo возвращает объект Repo
func NewRepo(path string) (*Repo, error) {
	if path == "" {
		return nil, &InternalError{
			Text:   "не указан путь к репозиторию",
			Caller: "Manager::NewRepoObj",
		}
	}
	repo := new(Repo)
	repo.path = path
	return repo, nil
}

// Path возвращает путь к репозиторию
func (r *Repo) Path() string {
	return r.path
}

// OpenDB открывает подключение к БД
func (r *Repo) OpenDB() error {
	fp := pathDB(r.path)
	if !FileExists(fp) {
		return &InternalError{
			Text:   "Репозиторий не инициализирован",
			Caller: "Manager::OpenDB",
		}
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
		return &InternalError{
			Text:   "Ошибка создания соединения с БД",
			Caller: "Manager::OpenDB",
			Err:    err,
		}
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
			fmt.Printf("ошибка оптимизации БД: %v\n", err)
		}
		if err = r.db.Close(); err != nil {
			return &InternalError{
				Text:   "ошибка закрытия БД",
				Caller: "Manager::Close",
				Err:    err,
			}
		}
		r.db = nil
	}
	return nil
}

// Clean - очистка и ре-индекс БД
func (r *Repo) Clean() error {
	if r.db != nil {
		if _, err = r.db.Exec("VACUUM"); err != nil {
			return &InternalError{
				Text:   "ошибка очистки БД",
				Caller: "Manager::Clean::vacuum",
				Err:    err,
			}
		}
		if _, err = r.db.Exec("REINDEX"); err != nil {
			return &InternalError{
				Text:   "ошибка перестройки идекса таблиц БД",
				Caller: "Manager::Clean::reindex",
				Err:    err,
			}
		}
	}
	return nil
}

// PackageID возвращает ID пакета
func (r *Repo) PackageID(pack string) (int64, error) {
	var id int64
	// if err = r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id); err == sql.ErrNoRows {
	if err = r.db.QueryRow("SELECT id FROM packages WHERE name=?;", pack).Scan(&id); err != nil {
		return 0, &InternalError{
			Text:   fmt.Sprintf("ошибка получения ID пакета %q", pack),
			Caller: "Manager::PackageID",
			Err:    err,
		}
	}
	return id, nil
}

// NewPackage создает запись нового пакета и возвращает его ID
func (r *Repo) NewPackage(name string) (id int64, err error) {
	res, err := r.db.Exec("INSERT INTO packages ('name', 'hash') VALUES (?, 0);", name)
	if err != nil {
		return 0, &InternalError{
			Text:   fmt.Sprintf("ошибка создания записи о пакете %q в БД", name),
			Caller: "Manager::NewPackageID",
			Err:    err,
		}
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
	pck := alias[0]
	als := alias[1]
	if !r.PackIsActive(pck) {
		return &InternalError{
			Text:   fmt.Sprintf("пакет %q не найден или заблокирован", pck),
			Caller: "Manager::SetAlias",
		}
	}
	if res, err := r.db.Exec("INSERT INTO aliases (name, alias) VALUES (?, ?);", pck, als); err != nil {
		switch err.(type) {
		case sqlite3.Error:
			if err.(sqlite3.Error).Code == sqlite3.ErrConstraint {
				return &InternalError{
					Text: fmt.Sprintf("Псевдоним для пакета %q уже задан", pck),
					Err:  err,
				}
			}
			return &InternalError{
				Text:   fmt.Sprintf("ошибка создания псевдонима %q для пакета %q", als, pck),
				Caller: "Manager::SetAlias::SQL",
				Err:    err,
			}
		default:
			return &InternalError{
				Text:   fmt.Sprintf("ошибка создания псевдонима %q для пакета %q", als, pck),
				Caller: "Manager::SetAlias" + fmt.Sprintf("[%v]", err.(sqlite3.Error).Code),
				Err:    err,
			}
		}
	} else if c, err := res.RowsAffected(); err != nil || c != 1 {
		return &InternalError{
			Text:   fmt.Sprintf("псевдоним не добавлен: count=%d;excpect: 1", c),
			Caller: "Manager::SetAlias",
			Err:    err,
		}
	}
	fmt.Printf("Установлен псевдоним: [ %v ]=( %v )\n", pck, als)
	return nil
}

// DelAlias удалает псевдоним
func (r *Repo) DelAlias(alias string) error {
	if res, err := r.db.Exec("DELETE FROM aliases WHERE alias=?;", alias); err != nil {
		return &InternalError{
			Text:   "ошибка удаления псевдонима",
			Caller: "Manager::DelAlias",
			Err:    err,
		}
	} else if c, _ := res.RowsAffected(); c != 1 {
		return &InternalError{
			Text:   fmt.Sprintf("не найден псевдоним %q", alias),
			Caller: "Manager::DelAlias",
			Err:    err,
		}
	}
	fmt.Printf("Удален псевдоним: [ %v ]\n", alias)
	return nil
}

// NoIndexedPacks возвращает список не проиндексированных пакетов
func (r *Repo) NoIndexedPacks() []string {
	lst := make([]string, 0)
	for _, pack := range r.ActivePacks() {
		if !r.PackIsIndexed(pack) {
			lst = append(lst, pack)
		}
	}
	return lst
}

// ActivePacks кэширует и возвращает список пакетов имеющихся в репозитории,
// за исключением заблокированных
func (r *Repo) ActivePacks() []string {
	if len(r.actPacks) == 0 {
		ch := make(chan string, 3)
		go DirList(r.path, ch)
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
func (r *Repo) DisabledPacks() []string {
	if len(r.disPacks) == 0 {
		rows, err := r.db.Query("SELECT name FROM excludes;")
		if err != nil {
			log.Fatalf("Manager::DisabledPacks: error select packs: %v", err)
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

// HashedPackages собирает данные о пакетах в БД в структуру HashedPackData
// и передает по каналу
func (r *Repo) HashedPackages(packs chan HashedPackData) error {
	defer close(packs)
	rows, err := r.db.Query("SELECT id, name, hash, exec FROM packages ORDER BY name;")
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return &InternalError{
			Text:   "ошибка выборки пакетов",
			Caller: "Manager::HashedPackages",
			Err:    err,
		}
	}
	defer rows.Close()

	var pData HashedPackData
	var filesPackDB []*FileInfo

	for rows.Next() {
		if err = rows.Scan(&pData.ID, &pData.Name, &pData.Hash, &pData.Exec); err != nil {
			return &InternalError{
				Text:   "ошибка выборки пакетов",
				Caller: "Manager::HashedPackages",
				Err:    err,
			}
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
func (r *Repo) FilesPackRepo(pack string) []*FileInfo {
	path := filepath.Join(r.path, pack)   // base Path repopath/packname
	fInfoList := make([]*FileInfo, 0, 50) // reserve place for ~50 files
	// unWanted, _ := regexp.Compile(`(.*[Tt]humb[s]?\.db)|(^~.*)`)
	fInfoCh := dirWalk(path)
	for fInfo := range fInfoCh {
		fi := new(FileInfo)
		*fi = fInfo

		// if unWanted.MatchString(fp) {
		// continue
		// } else {
		fInfoList = append(fInfoList, fi)
		// }
	}
	sort.Slice(fInfoList, func(i, j int) bool { return fInfoList[i].Path < fInfoList[j].Path })
	return fInfoList
}

// FilesPackDB возвращает список файлов пакета имеющихся в БД
func (r *Repo) FilesPackDB(id int64) ([]*FileInfo, error) {
	rows, err := r.db.Query("SELECT id, path, size, mdate, hash FROM files WHERE package_id=? ORDER BY path;", id)
	if err != nil {
		return nil, &InternalError{
			Text:   "ошибка выборки файлов",
			Caller: "Manager::FilesPackDB",
			Err:    err,
		}
	}
	defer rows.Close()
	var pFileInfoList []*FileInfo

	for rows.Next() {
		fd := new(FileInfo)
		if err := rows.Scan(&fd.ID, &fd.Path, &fd.Size, &fd.MDate, &fd.Hash); err != nil {
			return nil, &InternalError{
				Text:   "ошибка выборки файлов",
				Caller: "Manager::FilesPackDB::Scan",
				Err:    err,
			}
		}
		pFileInfoList = append(pFileInfoList, fd)
	}
	return pFileInfoList, nil
}

// PackIsIndexed определяет проиндексирован ли пакет
func (r *Repo) PackIsIndexed(name string) bool {
	for _, fn := range r.Packages() {
		if name == fn {
			return true
		}
	}
	return false
}

// PackIsBlocked проверка пакета на блокировку
func (r *Repo) PackIsBlocked(name string) bool {
	for _, fn := range r.DisabledPacks() {
		if name == fn {
			return true
		}
	}
	return false
}

// PackIsActive проверка отсутствия у пакета блокировки
func (r *Repo) PackIsActive(pack string) bool {
	for _, fp := range r.ActivePacks() {
		if fp == pack {
			return true
		}
	}
	return false
}

// AddFileData добавляет данные файла пакета в БД при обнаружении в репозитории
func (r *Repo) AddFileData(fInfo *FileInfo) error {
	if res, err := r.stmtAddFile.Exec(fInfo.ID, fInfo.Path, fInfo.Size, fInfo.MDate, fInfo.Hash); err != nil {
		return &InternalError{
			Text:   "ошибка добавления файла",
			Caller: "Manager::AddFile::stmtAddFile",
			Err:    err,
		}
	} else if ret, _ := res.RowsAffected(); ret == 0 {
		return &InternalError{
			Text:   "ошибка добавления файла",
			Caller: "Manager::AddFile::stmtAddFile",
			Err:    err,
		}
	}
	return nil
}

// UpdateFileData обновляет данные о файде в пакете при изменении в репозитории
func (r *Repo) UpdateFileData(fd *FileInfo) error {
	if res, err := r.stmtUpdFile.Exec(fd.Size, fd.MDate, fd.Hash, fd.ID); err != nil {
		return &InternalError{
			Text:   "ошибка обновления файла",
			Caller: "Manager::UpdateFileData::stmtUpdFile",
			Err:    err,
		}
	} else if ret, _ := res.RowsAffected(); ret == 0 {
		return &InternalError{
			Text:   "ошибка обновления файла",
			Caller: "Manager::UpdateFileData::stmtUpdFile",
			Err:    err,
		}
	}
	return nil
}

// RemoveFileData удаляет данные о файле из БД при отсутствии в репозитории
func (r *Repo) RemoveFileData(fInfo *FileInfo) error {
	if res, err := r.stmtDelFile.Exec(fInfo.ID); err != nil {
		return &InternalError{
			Text:   "ошибка обновления файла",
			Caller: "Manager::RemoveFile::stmtDelFile",
			Err:    err,
		}
	} else if ret, _ := res.RowsAffected(); ret == 0 {
		return &InternalError{
			Text:   "ошибка обновления файла",
			Caller: "Manager::RemoveFile::stmtDelFile",
			Err:    err,
		}
	}
	return nil
}

// CleanPacks сбрасывает кэш с данными об активных, индексированных и блокированных пакетах
// при команде об активации или блокировке
func (r *Repo) CleanPacks() error {
	// clean cached lists
	r.actPacks = []string{}
	r.disPacks = []string{}
	r.indPacks = []string{}
	for _, pack := range r.Packages() { // проход по списку пакетов в БД
		if !r.PackIsActive(pack) {
			if err = r.RemovePack(pack); err != nil {
				return err
			}
		}
	}
	return nil
}

// Packages возвращает список проиндексированных пакетов
func (r *Repo) Packages() []string {
	var packs []string
	var name string
	rows, _ := r.db.Query("SELECT name FROM packages ORDER BY Name;")
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&name)
		packs = append(packs, name)
	}
	return packs
}

// DisablePack блокирует пакет
func (r *Repo) DisablePack(pack string) error {

	res, err := r.db.Exec("INSERT INTO excludes VALUES (?);", pack)
	if err != nil {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка деактивации пакета %q", pack),
			Caller: "Manager::DisablePack",
			Err:    err,
		}
	}
	if c, _ := res.RowsAffected(); c != 1 {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка деактивации пакета %q", pack),
			Caller: "Manager::DisablePack::Count0",
			Err:    err,
		}
	}
	fmt.Printf("заблокирован: [ %s ]\n", pack)
	return nil
}

// EnablePack активирует пакет
func (r *Repo) EnablePack(pack string) error {
	res, err := r.db.Exec("DELETE FROM excludes WHERE Name=?;", pack)
	if err != nil {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка активации пакета %q", pack),
			Caller: "Manager::EnablePack",
			Err:    err,
		}
	}
	if c, _ := res.RowsAffected(); c != 1 {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка активации пакета %q", pack),
			Caller: "Manager::EnablePack::Count0",
			Err:    err,
		}
	}
	fmt.Printf("активирован: [ %s ]\n", pack)
	return nil
}

// RemovePack удаляет данные о пакете
func (r *Repo) RemovePack(pack string) error {
	res, err := r.db.Exec("DELETE FROM packages WHERE Name=?;", pack)
	if err != nil {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка удаления пакета %q", pack),
			Caller: "Manager::RemovePack",
			Err:    err,
		}
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return &InternalError{
			Text:   fmt.Sprintf("ошибка удаления пакета %q", pack),
			Caller: "Manager::RemovePack::Count0",
			Err:    err,
		}
	}
	fmt.Printf("  - [ %s ]\n", pack)
	return nil
}

// DBCleanPackages удаляет все данные о пакетах
func (r *Repo) DBCleanPackages() error {
	_, err := r.db.Exec("DELETE FROM packages;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка очистки данных пакетов",
			Caller: "Manager::DBCleanPackages",
			Err:    err,
		}
	}
	return nil
}

// DBCleanAliases удаляет все данные о псевдонимах
func (r *Repo) DBCleanAliases() error {
	_, err := r.db.Exec("DELETE FROM aliases;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка очистки данных псевдонимов",
			Caller: "Manager::DBCleanAliases",
			Err:    err,
		}
	}
	return nil
}

// DBCleanStatus удаляет все данные о блокировках
func (r *Repo) DBCleanStatus() error {
	_, err := r.db.Exec("DELETE FROM excludes;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка очистки данных блокировок",
			Caller: "Manager::DBCleanStatus",
			Err:    err,
		}
	}
	return nil
}

// HashSumPack подсчет контрольной суммы пакета по основанию сумм файлов
func (r *Repo) HashSumPack(id int64) error {
	var hash, hTotal string
	rows, _ := r.db.Query("SELECT hash FROM files WHERE package_id=?;", id)
	defer rows.Close()
	for rows.Next() {
		_ = rows.Scan(&hash)
		hTotal += hash
	}
	hash = HashSum(hTotal)
	res, err := r.db.Exec("UPDATE packages SET hash=? WHERE id=?;", hash, id)
	if err != nil {
		return &InternalError{
			Text:   "ошибка подсчета контрольной суммы",
			Caller: "Manager::HashSumPack",
			Err:    err,
		}
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return &InternalError{
			Text:   "ошибка подсчета контрольной суммы",
			Caller: "Manager::HashSumPack::Count0",
			Err:    err,
		}
	}
	return nil
}

// SetPrepare компилирует SQL шаблоны запросов для ускорения обработки данных
func (r *Repo) SetPrepare() error {
	//
	sqlExpr := "INSERT INTO files ('package_id', 'path', 'size', 'mdate', 'hash') VALUES (?, ?, ?, ?, ?);"
	r.stmtAddFile, err = r.db.Prepare(sqlExpr)
	if err != nil {
		return &InternalError{
			Text:   "ошибка подготовки данных запроса",
			Caller: "Manager::SetPrepare::insert",
			Err:    err,
		}
	}
	//
	r.stmtDelFile, err = r.db.Prepare("DELETE FROM files WHERE id=?;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка подготовки данных запроса",
			Caller: "Manager::SetPrepare::delete",
			Err:    err,
		}
	}
	//
	r.stmtUpdFile, err = r.db.Prepare("UPDATE files SET size=?, mdate=?, hash=? WHERE id=?;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка подготовки данных запроса",
			Caller: "Manager::SetPrepare::update",
			Err:    err,
		}
	}
	return nil
}

// Status выводит информацию о состоянии репозитория
func (r *Repo) Status() (*RepoStData, error) {
	data := new(RepoStData)
	// количество активных пакетов
	if err = r.db.QueryRow("SELECT COUNT() FROM packages;").Scan(&data.IndexedCnt); err != nil {
		return nil, &InternalError{
			Text:   "ошибка запроса данных в БД",
			Caller: "Manager::Status::active",
			Err:    err,
		}
	}

	// количество заблокированных
	if err = r.db.QueryRow("SELECT COUNT() FROM excludes;").Scan(&data.BlockedCnt); err != nil {
		return nil, &InternalError{
			Text:   "ошибка запроса данных в БД",
			Caller: "Manager::Status::disabled",
			Err:    err,
		}
	}

	// количество пакетов в репозитории
	ch := make(chan string)
	go DirList(r.Path(), ch)
	for range ch {
		data.TotalCnt++
	}

	// данные БД файла
	fInfo, err := os.Stat(pathDB(r.Path()))
	if err != nil {
		data.DBSize = -1
		data.DBMDate = time.Time{}
	} else {
		data.DBSize = fInfo.Size()
		data.DBMDate = fInfo.ModTime()
	}

	// данные индекс файла
	fInfo, err = os.Stat(filepath.Join(r.Path(), IndexGZ))
	if err != nil {
		data.IndexSize = -1
		data.IndexMDate = time.Time{}
	} else {
		data.IndexSize = fInfo.Size()
		data.IndexMDate = fInfo.ModTime()
	}

	// данные индекс-хэш файла
	fInfo, err = os.Stat(filepath.Join(r.Path(), IndexGZ+".sha1"))
	if err != nil {
		data.HashSize = -1
		data.HashMDate = time.Time{}
	} else {
		data.HashSize = fInfo.Size()
		data.HashMDate = fInfo.ModTime()
	}
	return data, nil
}

// List формирует и передает поканалу список проиндексированных пакетов с данными о статусе
func (r *Repo) List(ch chan<- *ListData) {
	dir := make(chan string)
	go DirList(r.Path(), dir)
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
			data.Status = PackStatusBlocked
		} else if r.PackIsIndexed(name) {
			// актл
			data.Status = PackStatusActive
		} else {
			// !инд
			data.Status = PackStatusNotIndexed
		}
		ch <- data
	}
	close(ch)
}

// checkDB проверяет целостность БД
func (r *Repo) checkDB() error {
	rows, err := r.db.Query("PRAGMA integrity_check;")
	if err != nil {
		return &InternalError{
			Text:   "ошибка проверки целостности БД",
			Caller: "Manager::checkDB",
			Err:    err,
		}
	}
	var res interface{}
	for rows.Next() {
		err = rows.Scan(&res)
		if res != "ok" || err != nil {
			return &InternalError{
				Text:   "ошибка целостности БД. Tребуется повторная инициализация",
				Caller: "Manager::checkDB",
				Err:    err,
			}
		}
	}
	return nil
}

// CheckDBVersion проверка соответствия версии БД репозитория перед индексацией
func (r *Repo) CheckDBVersion() error {
	vmaj, vmin, err := r.VersionDB()
	if err != nil {
		return err
	} // todo - миграцию при поднятии версии программы
	if DBVersionMajor > vmaj {
		return &InternalError{
			Text:   "\n\tУстаревшая версия репозитория. Требуется переиндексация",
			Caller: "Manager::CheckDBVersion",
		}
	} else if DBVersionMinor > vmin {
		return &InternalError{
			Text:   "\n\tТребуется миграция БД репозитория",
			Caller: "Manager::CheckDBVersion",
		}
	} else if vmaj > DBVersionMajor || vmin > DBVersionMinor {
		msg := "\n\tверсия БД [%d.%d] старше требуемой[%d.%d]; возможно вы используете старую версию программы"
		return &InternalError{
			Text:   fmt.Sprintf(msg, vmaj, vmin, DBVersionMajor, DBVersionMinor),
			Caller: "Manager::CheckDBVersion",
		}
	}
	return nil
}

// EmptyExecFilesList возвращает список пакетов с пустыми значениями данных об исполняемых файлах
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

// VersionDB возвращает версию структуры БД
func (r *Repo) VersionDB() (int64, int64, error) {
	var vmaj, vmin int64
	err := r.db.QueryRow("SELECT vers_major, vers_minor FROM info WHERE id=1;").Scan(&vmaj, &vmin)
	if err != nil {
		return 0, 0, &InternalError{
			Text:   "нет данных о версии БД. Требуется инициализация",
			Caller: "Manager::VersionDB",
		}
	}
	return vmaj, vmin, nil
}

// ExecFileSet фиксирует имя исполняемого файла пакета
func (r *Repo) ExecFileSet(pack string, force bool) error {
	id, err := r.PackageID(pack)
	if err != nil {
		return err
	}

	switch force { // force - принудительная замена
	case false:
		var execInDB string
		if err := r.db.QueryRow(
			"SELECT CASE WHEN exec IS null THEN '' ELSE exec END exec FROM packages WHERE id=?;", id).Scan(
			&execInDB); err != nil {
			//if err := r.db.QueryRow("SELECT exec FROM packages WHERE id=?;", id).Scan(&exec_db); err != nil {
			return &InternalError{
				Text:   "ошибка запроса данных в БД",
				Caller: "Manager::ExecFileSet",
				Err:    err,
			}
		}
		if execInDB != "" {
			fmt.Printf("\t%v: уже установлен, пропуск\n", pack)
			return nil
		}
		fallthrough
	case true:
		execFile, err := defineExecFile(r, pack)
		if err != nil {
			return err
		}
		res, err := r.db.Exec("UPDATE packages SET exec=? WHERE id=?;", execFile, id)
		if err != nil {
			return &InternalError{
				Text:   "ошибка обновления данных в БД",
				Caller: "Manager::ExecFileSet",
				Err:    err,
			}
		}
		if c, _ := res.RowsAffected(); c == 0 {
			return &InternalError{
				Text:   "ошибка обновления данных в БД",
				Caller: "Manager::ExecFileSet::Count0",
				Err:    err,
			}
		}
		fmt.Printf("\t%v: установлен в [ %v ]\n", pack, execFile)
	}
	return nil
}

// ExecFileDel удаляет информацию об исполняемом файле пакета
func (r *Repo) ExecFileDel(pack string) error {
	id, err := r.PackageID(pack)
	if err != nil {
		return err
	}
	// CheckError(fmt.Sprintf("не найден пакет в БД: %v", pack), &err)
	res, err := r.db.Exec("UPDATE packages SET exec='noexec' WHERE id=?;", id)
	if err != nil {
		return &InternalError{
			Text:   "ошибка удаления данных в БД",
			Caller: "Manager::ExecFileDel",
			Err:    err,
		}
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return &InternalError{
			Text:   "ошибка удаления данных в БД",
			Caller: "Manager::ExecFileDel::Count0",
			Err:    err,
		}
	}
	fmt.Printf("Исполняемый файл пакета '%v' установлен в 'noexec' \n", pack)
	return nil
}

// ExecFileInfo возвращает информацию об исполняемом файле пакета
func (r *Repo) ExecFileInfo(pack string) (string, error) {
	id, err := r.PackageID(pack)
	if err != nil {
		return "", err
	}
	// CheckError(fmt.Sprintf("не найден пакет в БД: %v", pack), &err)
	var execInDB string

	if err := r.db.QueryRow(
		"SELECT CASE WHEN exec IS null THEN '' ELSE exec END exec FROM packages WHERE id=?;", id).Scan(
		&execInDB); err != nil {
		//if err := r.db.QueryRow("SELECT exec FROM packages WHERE id=?;", id).Scan(&exec_db); err != nil {
		return "", &InternalError{
			Text:   "ошибка запроса данных в БД",
			Caller: "Manager::ExecFileInfo",
			Err:    err,
		}
	}
	return execInDB, nil
}

// CheckEmptyExecFiles проверяет на наличие не установленных исполняемых файлах пакетов в репозитории
func (r *Repo) CheckEmptyExecFiles() error {
	if len(r.EmptyExecFilesList()) > 0 {
		return &InternalError{
			Text: "\n\tТребуется определить исполняемые файлы\n\t" +
				"Запустите программу с командой 'exec check'\n",
			Caller: "Manager::CheckEmptyExecFiles",
		}
	}
	return nil
}
