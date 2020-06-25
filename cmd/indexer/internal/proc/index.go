package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, fullmode bool, packs []string) {
	err = r.CheckDBVersion()
	utils.CheckError("", &err)
	const errIndexMsg = errMsg + ":index:"
	var changed bool

	CheckRegl(r.Path())
	for _, pack := range packs {
		if r.PackIsBlocked(pack) {
			panic(fmt.Sprintf("пакет [ %v ] заблокирован", pack))
		}
	}
	err = r.SetPrepare()
	utils.CheckError(fmt.Sprintf("%v:setprepare:", errIndexMsg), &err)

	for _, pack := range packs {
		if pack == "" {
			panic("задано пустое имя пакета")
		}
		fmt.Println("[", pack, "]")
		if processPackIndex(r, fullmode, pack) && !changed {
			changed = true
		}

	}
	fmt.Println()
	err = r.CleanPacks()
	utils.CheckError(fmt.Sprintf("%v:cleanpacks:", errIndexMsg), &err)

	obj.ShowEmptyExecFiles(r)

	if changed {
		fmt.Println(doPopMsg)
	} else {
		fmt.Println(noChangeMsg)
	}
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, fullmode bool, pack string) bool {
	const errPackIndMsg = errMsg + ":index::processPackIndex:"
	var (
		packID                   int64           // ID пакета
		fsPath                   string          // file path on fs
		fsList                   []string        // список файлов пакета в репозитории
		dbData                   *obj.FileInfo   // db file object
		pFileInfo                *obj.FileInfo   //
		dbList                   []*obj.FileInfo // список файлов пакета в БД
		fsInd, dbInd             int             //  counters
		packChanged, fileChanged bool            // package has changes
		err                      error
	)

	packID, err = r.GetPackageID(pack)
	if err != nil { // нет такого пакета
		packID, err = r.NewPackage(pack)
		utils.CheckError(fmt.Sprintf("%v:%v:", errPackIndMsg, "NewPackage"), &err)
	}

	pFileInfo = new(obj.FileInfo)
	fsList, err = r.FilesPackRepo(pack)
	utils.CheckError("", &err)

	dbList, err = r.FilesPackDB(packID)
	utils.CheckError(fmt.Sprintf("%v:%v:", errPackIndMsg, "FilesPackDB"), &err)

	fsMaxInd := len(fsList) - 1
	// fDBCount, err := r.FilesPackDBCount(packID)
	// utils.CheckError(fmt.Sprintf("%v:%v:", errPackIndMsg, "get files count from DB"), &err)
	dbMaxInd := len(dbList) - 1
	sort.Slice(fsList, func(i, j int) bool { return fsList[i] < fsList[j] })
	// sort.Slice(dbList, func(i, j int) bool { return dbList[i].Path < dbList[j].Path })

	for {
		// завершили обход списков
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}
		// no in BD: add file to BD
		if dbInd > dbMaxInd {
			// добавляем запись о файле в БД
			fsPath = fsList[fsInd]
			err = r.AddFile(packID, pack, fsPath)
			utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)
			fmt.Println("  +", fsPath)
			//next path in FS list
			fsInd++
			if !packChanged {
				packChanged = true
			}
			continue
		}
		// удаляем запись о файле из БД
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			err = r.RemoveFile(dbData.ID)
			utils.CheckError(fmt.Sprintf("%v:error compare files", errPackIndMsg), &err)
			fmt.Println("  -", dbData.Path)
			// next file obj in db list
			dbInd++
			continue
		}

		fsPath = fsList[fsInd]
		dbData = dbList[dbInd]

		// сверка данных о файле в БД и в репозитории
		// in FS, in db
		if fsPath == dbData.Path {
			fp := filepath.Join(r.Path(), pack, fsPath)

			fInfo, err := os.Stat(fp)
			if err != nil {
				utils.CheckError(fmt.Sprintf("%v:error stat files", errPackIndMsg), &err)
			}

			fileChanged = !(fInfo.Size() == dbData.Size && fInfo.ModTime().UnixNano() == dbData.MDate)

			if fullmode || fileChanged {
				hash, err := utils.HashSumFile(fp)
				if err != nil {
					utils.CheckError(fmt.Sprintf("%v:error hashsum calculate", errPackIndMsg), &err)
				}

				pFileInfo.ID = dbData.ID
				pFileInfo.Path = fp
				pFileInfo.Size = fInfo.Size()
				pFileInfo.MDate = fInfo.ModTime().UnixNano()
				pFileInfo.Hash = hash

				err = r.UpdateFileData(pFileInfo)
				utils.CheckError(fmt.Sprintf("%v:error update file data in DB", errPackIndMsg), &err)

				fmt.Println("  .", dbData.Path)

				if !packChanged {
					packChanged = true
				}
			}
			fsInd++
			dbInd++
			// in FS, not in db: add file to BD
		} else if fsPath < dbData.Path {
			// добавляем запись о файле в БД
			err = r.AddFile(packID, pack, fsPath)
			utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)

			fmt.Println("  +", fsPath)

			if !packChanged {
				packChanged = true
			}
			fsInd++
			// удаляем запись о файле из БД
			// not in FS, in db
		} else if fsPath > dbData.Path {
			err = r.RemoveFile(dbData.ID)
			utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)

			fmt.Println("  -", dbData.Path)

			if !packChanged {
				packChanged = true
			}
			dbInd++
		} else {
			panic(fmt.Sprintf("%v something goes wrong", errPackIndMsg))
		}
		continue
	}

	// пересчитываем контрольную сумму пакета при наличии изменений файлов
	if packChanged {
		err = r.HashSumPack(packID)
		utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)
	}
	return packChanged
}
