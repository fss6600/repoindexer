package proc

import (
	"errors"
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"log"
	"sort"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, packs []string) error {
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
	for _, pack := range packs {
		if pack == "" {
			return errors.New("error: задано пустое имя пакета")
		}
		fmt.Println("[", pack, "]")
		if err := processPackIndex(r, pack); err != nil {
			return err
		}
	}
	r.CleanPacks()
	return nil
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) error {
	var (
		packID       int64          // ID пакета
		fsList       []string       // список файлов пакета в репозитории
		dbList       []obj.FileInfo // список файлов пакета в БД
		err          error
		fsInd, dbInd int          //  counters
		fsData       string       // file path on fs
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

	//fmt.Println("FS:", fsMaxInd, "; db:", dbMaxInd)

	sort.Slice(fsList, func(i, j int) bool { return fsList[i] < fsList[j] })
	sort.Slice(dbList, func(i, j int) bool { return dbList[i].Path < dbList[j].Path })

	for {
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}
		if dbInd > dbMaxInd { // no in db
			fsData = fsList[fsInd]
			fmt.Print(": calculate sum/date; ")
			fmt.Println(":1: add to db")
			fsInd++ //next path in FS list
			if !changed {
				changed = true
			}
			continue
		}
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			fmt.Print("dbData: ", dbData.Path, ":\t")
			fmt.Println(dbData.Path, ":1: del from db")
			dbInd++ // next file obj in db list
			continue
		}

		fsData = fsList[fsInd]
		dbData = dbList[dbInd]

		if fsData == dbData.Path { // in FS, in db
			fmt.Print(fsData, "=", dbData.Path)
			_, err := internal.CheckSums(fsData)
			if err != nil {
				log.Fatal(err)
			}
			fsInd++
			dbInd++

		} else if fsData < dbData.Path { // in FS, not in db
			fmt.Print(fsData)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(":2: add to db")
			if !changed {
				changed = true
			}
			fsInd++
		} else if fsData > dbData.Path { // not in FS, in db
			fmt.Println(dbData.Path, ":2: dell from db")
			dbInd++
		} else {
			log.Fatal("wrong")
		}
		continue
	}
	return nil
}
