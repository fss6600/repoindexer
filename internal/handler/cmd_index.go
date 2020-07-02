package handler

import (
	"fmt"
	"path/filepath"
)

// Index обработка команды `index` - индексация пакетов в репозитории
// без параметров выполняет полную индексацию репозитория
// при указании имен пакетов, выполняет индексацию указанных
// имена пакетов содержащие пробел следует передавать в кавычках
func Index(r *Repo, fullmode bool, packs []string) error {
	if err = r.CheckDBVersion(); err != nil {
		return err
	}
	// флаг наличия изменений в пакете
	var changed bool
	// проверка установки режима регламента
	if err = checkRegl(r.path); err != nil {
		return err
	}
	// проверка пакета на блокировку
	for _, pack := range packs {
		if r.PackIsBlocked(pack) {
			return &internalError{
				Text:   fmt.Sprintf("пакет %q заблокирован", pack),
				Caller: "Index",
			}
		}
	}
	// проверка на готовность БД
	if err = r.SetPrepare(); err != nil {
		return err
	}

	for _, pack := range packs {
		if pack == "" {
			return &internalError{
				Text:   "задано пустое имя пакета",
				Caller: "Index",
			}
		}
		fmt.Println("[", pack, "]")
		done, err := processPackIndex(r, fullmode, pack)
		if err != nil {
			return err
		}
		if done && !changed {
			changed = true
		}
	}
	fmt.Println()
	if err = r.CleanPacks(); err != nil {
		return err
	}

	// вывод данных о неустановленных исполняемых файлах
	ShowEmptyExecFiles(r)

	if changed {
		fmt.Println(doPopMsg)
	} else {
		fmt.Println(noChangeMsg)
	}
	return nil
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *Repo, fullmode bool, pack string) (bool, error) {
	var (
		packID                   int64       // ID пакета
		dbData, fInfo            *FileInfo   // указатель на объект с данными файла
		fsList, dbList           []*FileInfo // список файлов пакета в БД
		fsInd, dbInd             int         // counters
		packChanged, fileChanged bool        // package has changes
		err                      error       // error
		fpRel                    string      // путь к файлу относительно пакета
	)

	packID, err = r.PackageID(pack)
	if err != nil { // нет такого пакета
		if packID, err = r.NewPackage(pack); err != nil {
			return false, err
		}
	}

	fsList = r.FilesPackRepo(pack)
	if dbList, err = r.FilesPackDB(packID); err != nil {
		return false, err
	}

	fsMaxInd := len(fsList) - 1
	dbMaxInd := len(dbList) - 1

	for {
		// завершили обход списков
		if fsInd > fsMaxInd && dbInd > dbMaxInd { // end both lists
			break
		}

		// Вариант1: Новый пакет или пакет удален
		// пакета нет в БД - добавить
		if dbInd > dbMaxInd {
			// добавляем запись о файле в БД
			fInfo = fsList[fsInd]
			fpRel, _ = filepath.Rel(filepath.Join(r.Path(), pack), fInfo.Path)

			fInfo.ID = packID
			if fInfo.Hash, err = getFileHash(fInfo.Path); err != nil {
				return false, err
			}
			fInfo.Path = fpRel

			if err = r.AddFile(fInfo); err != nil {
				return false, err
			}

			fmt.Println("  +", fInfo.Path)
			fsInd++
			if !packChanged {
				packChanged = true
			}
			//next path in FS list
			continue
		}

		// пакета нет в репозитории - удаляем запись о файле из БД
		if fsInd > fsMaxInd { // not in FS
			dbData = dbList[dbInd]
			if err = r.RemoveFile(dbData); err != nil {
				return false, err
			}
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
		fpRel, _ = filepath.Rel(filepath.Join(r.path, pack), fInfo.Path)

		// Вариант2: Данные пакета изменились
		// сверка данных о файле в БД и в репозитории
		// in FS, in db
		if fpRel == dbData.Path {
			fileChanged = !(fInfo.Size == dbData.Size && fInfo.MDate == dbData.MDate)

			if fullmode || fileChanged {
				dbData.Size = fInfo.Size
				dbData.MDate = fInfo.MDate
				if dbData.Hash, err = getFileHash(fInfo.Path); err != nil {
					return false, err
				}

				if err = r.UpdateFileData(dbData); err != nil {
					return false, err
				}

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
			if fInfo.Hash, err = getFileHash(fInfo.Path); err != nil {
				return false, err
			}
			fInfo.Path = fpRel

			if err = r.AddFile(fInfo); err != nil {
				return false, err
			}

			fmt.Println("  +", fpRel)

			if !packChanged {
				packChanged = true
			}
			fsInd++

			// удаляем запись о файле из БД
			// not in FS, in db
		} else if fpRel > dbData.Path {
			if err = r.RemoveFile(dbData); err != nil {
				return false, err
			}

			fmt.Println("  -", dbData.Path)

			if !packChanged {
				packChanged = true
			}
			dbInd++
		} else {
			return false, &internalError{
				Text:   "что-то пошло не так при обходе списков файлов",
				Caller: "Index",
			}

		}
		continue
	}

	// пересчитываем контрольную сумму пакета при наличии изменений файлов
	if packChanged {
		if err = r.HashSumPack(packID); err != nil {
			return false, err
		}
	}
	return packChanged, nil
}

func getFileHash(fPath string) (string, error) {
	hash, err := HashSumFile(fPath)
	if err != nil {
		return "", err
	}
	return hash, nil
}
