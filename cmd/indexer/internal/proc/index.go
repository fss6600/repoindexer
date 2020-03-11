package proc

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, packs []string) (err error) {
	if len(packs) == 0 {
		// получает актуальные активные пакеты в репозитории
		packs = r.ActivePacks()
	} else {
		// проверяем актуальность (правильность) указанных пакетов
		for _, pack := range packs {
			if !r.IsActive(pack) {
				return fmt.Errorf("пакет [ %v ] не найден в репозитории или заблокирован", pack)
			}
		}
	}
	//fmt.Println(packs)
	if err = r.SetPrepare(); err != nil {
		return err
	}

	for _, pack := range packs {
		if pack == "" {
			return errors.New("error: задано пустое имя пакета")
		}
		fmt.Println("[", pack, "]")
		if err = processPackIndex(r, pack); err != nil {
			return err
		}
	}
	r.CleanPacks()
	return
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) error {
	var (
		packID       int64          // ID пакета
		fsList       []string       // список файлов пакета в репозитории
		dbList       []obj.FileInfo // список файлов пакета в БД
		err          error
		fsInd, dbInd int          //  counters
		fsPath       string       // file path on fs
		dbData       obj.FileInfo // db file object
		changed      bool         // package has changes
	)
	fsList, err = r.FilesPackRepo(pack)
	if err != nil {
		return err
	}
	packID = r.PackageID(pack)
	dbList, err = r.FilesPackDB(packID)
	if err != nil {
		return err
	}

	fsMaxInd := len(fsList) - 1
	dbMaxInd := len(dbList) - 1

	sort.Slice(fsList, func(i, j int) bool { return fsList[i] < fsList[j] })
	sort.Slice(dbList, func(i, j int) bool { return dbList[i].Path < dbList[j].Path })

	for {
		// завершили обход списков
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}
		if dbInd > dbMaxInd { // no in BD: add file to BD
			// добавляем запись о файле в БД
			fsPath = fsList[fsInd]
			if err := r.AddFile(packID, pack, fsPath); err != nil {
				return err
			}
			fmt.Println("+", fsPath)
			fsInd++ //next path in FS list
			if !changed {
				changed = true
			}
			continue
		}
		// удаляем запись о файле из БД
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			if err := r.RemoveFile(dbData.Id); err != nil {
				return err
			}
			fmt.Println("-", dbData.Path)
			dbInd++ // next file obj in db list
			continue
		}

		fsPath = fsList[fsInd]
		dbData = dbList[dbInd]

		// сверка данных о файле в БД и в репозитории
		if fsPath == dbData.Path { // in FS, in db
			res, err := r.ChangedFile(pack, fsPath, dbData)
			if err != nil {
				return fmt.Errorf("error compare files: %v", err)
			}
			if res {
				fmt.Println(".", dbData.Path)
				if !changed {
					changed = true
				}
			}
			fsInd++
			dbInd++

		} else if fsPath < dbData.Path { // in FS, not in db: add file to BD
			// добавляем запись о файле в БД
			if err := r.AddFile(packID, pack, fsPath); err != nil {
				return err
			}
			fmt.Println("+", fsPath)
			if !changed {
				changed = true
			}
			fsInd++
			// удаляем запись о файле из БД
		} else if fsPath > dbData.Path { // not in FS, in db
			if err := r.RemoveFile(dbData.Id); err != nil {
				return err
			}
			fmt.Println("-", dbData.Path)
			if !changed {
				changed = true
			}
			dbInd++
		} else {
			log.Fatal("wrong")
		}
		continue
	}

	// пересчитываем контрольную сумму пакета при наличии изменений файлов
	if changed {
		if err := r.HashSumPack(packID); err != nil {
			return err
		}
	}

	return nil
}
