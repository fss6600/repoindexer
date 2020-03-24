package proc

import (
	"fmt"
	"sort"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, packs []string) {
	const tmplErrMsg = "error::index:"
	CheckRegl(r.Path())
	if len(packs) == 0 {
		packs = r.ActivePacks()
	} else {
		for _, pack := range packs {
			if r.PackIsBlocked(pack) {
				panic(fmt.Sprintf("пакет [ %v ] заблокирован", pack))
			}
		}
	}
	err = r.SetPrepare()
	utils.CheckError(fmt.Sprintf("%v:setprepare:", tmplErrMsg), &err)

	for _, pack := range packs {
		if pack == "" {
			panic("задано пустое имя пакета")
		}
		fmt.Println("[", pack, "]")
		processPackIndex(r, pack)
	}
	err := r.CleanPacks()
	utils.CheckError(fmt.Sprintf("%v:cleanpacks:", tmplErrMsg), &err)
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) {
	const tmplErrMsg = "error::index::processPackIndex:"
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
	utils.CheckError(fmt.Sprintf("%v:%v:", tmplErrMsg, "FilesPackRepo"), &err)
	packID, err = r.PackageID(pack)
	utils.CheckError(fmt.Sprintf("%v:%v:", tmplErrMsg, "PackageID"), &err)
	dbList, err = r.FilesPackDB(packID)
	utils.CheckError(fmt.Sprintf("%v:%v:", tmplErrMsg, "FilesPackDB"), &err)

	fsMaxInd := len(fsList) - 1
	dbMaxInd := len(dbList) - 1

	sort.Slice(fsList, func(i, j int) bool { return fsList[i] < fsList[j] })
	sort.Slice(dbList, func(i, j int) bool { return dbList[i].Path < dbList[j].Path })

	for {
		// завершили обход списков
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}
		// no in BD: add file to BD
		if dbInd > dbMaxInd {
			// добавляем запись о файле в БД
			fsPath = fsList[fsInd]
			err := r.AddFile(packID, pack, fsPath)
			utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
			fmt.Println("+", fsPath)
			//next path in FS list
			fsInd++
			if !changed {
				changed = true
			}
			continue
		}
		// удаляем запись о файле из БД
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			err := r.RemoveFile(dbData.Id)
			utils.CheckError(fmt.Sprintf("%v:error compare files", tmplErrMsg), &err)
			fmt.Println("-", dbData.Path)
			// next file obj in db list
			dbInd++
			continue
		}

		fsPath = fsList[fsInd]
		dbData = dbList[dbInd]

		// сверка данных о файле в БД и в репозитории
		// in FS, in db
		if fsPath == dbData.Path {
			res, err := r.ChangedFile(pack, fsPath, dbData)
			utils.CheckError(fmt.Sprintf("%v:error compare files", tmplErrMsg), &err)
			if res {
				fmt.Println(".", dbData.Path)
				if !changed {
					changed = true
				}
			}
			fsInd++
			dbInd++
			// in FS, not in db: add file to BD
		} else if fsPath < dbData.Path {
			// добавляем запись о файле в БД
			err := r.AddFile(packID, pack, fsPath)
			utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
			fmt.Println("+", fsPath)
			if !changed {
				changed = true
			}
			fsInd++
			// удаляем запись о файле из БД
			// not in FS, in db
		} else if fsPath > dbData.Path {
			err := r.RemoveFile(dbData.Id)
			utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
			fmt.Println("-", dbData.Path)
			if !changed {
				changed = true
			}
			dbInd++
		} else {
			panic(fmt.Sprintf("%v something goes wrong", tmplErrMsg))
		}
		continue
	}

	// пересчитываем контрольную сумму пакета при наличии изменений файлов
	if changed {
		err := r.HashSumPack(packID)
		utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
	}
}
