package proc

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const (
	errIndexMsg   = errMsg + ":index:"
	errPackIndMsg = errIndexMsg + ":processPackIndex:"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, fullmode bool, packs []string) {
	err = r.CheckDBVersion()
	utils.CheckError("", &err)
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
	var (
		packID                   int64           // ID пакета
		dbData, fInfo            *obj.FileInfo   //
		fsList, dbList           []*obj.FileInfo // список файлов пакета в БД
		fsInd, dbInd             int             //  counters
		packChanged, fileChanged bool            // package has changes
		err                      error
		fpRel                    string
	)

	packID, err = r.GetPackageID(pack)
	if err != nil { // нет такого пакета
		packID, err = r.NewPackage(pack)
		utils.CheckError(fmt.Sprintf("%v:%v:", errPackIndMsg, "NewPackage"), &err)
	}

	fsList, err = r.FilesPackRepo(pack)
	utils.CheckError("", &err)

	dbList, err = r.FilesPackDB(packID)
	utils.CheckError(fmt.Sprintf("%v:%v:", errPackIndMsg, "FilesPackDB"), &err)

	fsMaxInd := len(fsList) - 1
	dbMaxInd := len(dbList) - 1
	// fsListNotEmpty := fsMaxInd >= 0
	// dbListNotEmpty := dbMaxInd >= 0

	sort.Slice(fsList, func(i, j int) bool { return fsList[i].Path < fsList[j].Path })

	for {
		// завершили обход списков
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}

		// Вариант1: Новый пакет или пакет удален
		// пакета нет в БД: add file to BD
		if dbInd > dbMaxInd {
			// добавляем запись о файле в БД
			fInfo = fsList[fsInd]
			fpRel, _ = filepath.Rel(filepath.Join(r.Path(), pack), fInfo.Path)

			fInfo.ID = packID
			fInfo.Hash = getFileHash(fInfo.Path)
			fInfo.Path = fpRel

			err = r.AddFile(fInfo)
			utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)

			fmt.Println("  +", fInfo.Path)
			fsInd++
			if !packChanged {
				packChanged = true
			}
			//next path in FS list
			continue
		}

		// пакета нет в репо - удаляем запись о файле из БД
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			err = r.RemoveFile(dbData)
			utils.CheckError(fmt.Sprintf("%v:error compare files", errPackIndMsg), &err)
			fmt.Println("  -", dbData.Path)
			// next file obj in db list
			dbInd++
			if !packChanged {
				packChanged = true
			}
			//next path in DB list
			continue
		}

		dbData = dbList[dbInd]
		fInfo = fsList[fsInd]
		fpRel, _ = filepath.Rel(filepath.Join(r.Path(), pack), fInfo.Path)

		// Вариант2: Данные пакета изменились
		// сверка данных о файле в БД и в репозитории
		// in FS, in db
		if fpRel == dbData.Path {
			fileChanged = !(fInfo.Size == dbData.Size && fInfo.MDate == dbData.MDate)

			if fullmode || fileChanged {
				dbData.Size = fInfo.Size
				dbData.MDate = fInfo.MDate
				dbData.Hash = getFileHash(fInfo.Path)

				err = r.UpdateFileData(dbData)
				utils.CheckError(fmt.Sprintf("%v:error update file data in DB", errPackIndMsg), &err)

				fmt.Println("  .", fpRel)

				if !packChanged {
					packChanged = true
				}
			}
			fsInd++
			dbInd++

			// in FS, not in db: add file to BD
		} else if fpRel < dbData.Path {
			// добавляем запись о файле в БД
			fInfo.ID = packID
			fInfo.Hash = getFileHash(fInfo.Path)
			fInfo.Path = fpRel

			err = r.AddFile(fInfo)
			utils.CheckError(fmt.Sprintf("%v", errPackIndMsg), &err)

			fmt.Println("  +", fpRel)

			if !packChanged {
				packChanged = true
			}
			fsInd++

			// удаляем запись о файле из БД
			// not in FS, in db
		} else if fpRel > dbData.Path {
			err = r.RemoveFile(dbData)
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

func getFileHash(fPath string) string {
	hash, err := utils.HashSumFile(fPath)
	utils.CheckError(fmt.Sprintf("%v:error hashsum calculate", errPackIndMsg), &err)
	return hash
}
