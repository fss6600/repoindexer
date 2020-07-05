package handler

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

// Index обработка команды `index` - индексация пакетов в репозитории
// без параметров выполняет полную индексацию репозитория
// при указании имен пакетов, выполняет индексацию указанных
// имена пакетов содержащие пробел следует передавать в кавычках
func Index(r *Repo, fullmode bool, packs []string) error {
	if err = r.checkDBVersion(); err != nil {
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
		if r.packIsBlocked(pack) {
			return &InternalError{
				Text:   fmt.Sprintf("пакет %q заблокирован", pack),
				Caller: "Index",
			}
		}
	}
	// проверка на готовность БД
	if err = r.setPrepare(); err != nil {
		return err
	}

	for _, pack := range packs {
		if pack == "" {
			return &InternalError{
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
	if err = r.cleanPacks(); err != nil {
		return err
	}

	// вывод данных о неустановленных исполняемых файлах
	showEmptyExecFiles(r)

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
		fsInd, dbInd             int         // counters
		packID                   int64       // ID пакета
		dbData, fInfo            *FileInfo   // указатель на объект с данными файла
		fsList, dbList           []*FileInfo // список файлов пакета в БД
		packChanged, fileChanged bool        // package has changes
		fpRel                    string      // путь к файлу относительно пакета
	)

	packID, err = r.packageID(pack)
	if err != nil && err.(*InternalError).Err == sql.ErrNoRows { // нет такого пакета
		if packID, err = r.newPackage(pack); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	}

	fsList = r.filesPackRepo(pack)
	if dbList, err = r.filesPackDB(packID); err != nil {
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
			fInfo.ID = packID
			if fInfo.Hash, err = getFileHash(fInfo.Path); err != nil {
				return false, err
			}
			fpRel, _ = filepath.Rel(filepath.Join(r.path, pack), fInfo.Path)
			fInfo.Path = fpRel

			if err = r.addFileData(fInfo); err != nil {
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
			if err = r.removeFileData(dbData); err != nil {
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

				if err = r.updateFileData(dbData); err != nil {
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

			if err = r.addFileData(fInfo); err != nil {
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
			if err = r.removeFileData(dbData); err != nil {
				return false, err
			}

			fmt.Println("  -", dbData.Path)

			if !packChanged {
				packChanged = true
			}
			dbInd++
		} else {
			msg := "что-то пошло не так при обходе списков файлов пакета %s"
			return false, &InternalError{
				Text:   fmt.Sprintf(msg, pack),
				Caller: "Index",
			}
		}
		continue
	}

	// пересчитываем контрольную сумму пакета при наличии изменений файлов
	if packChanged {
		if err = r.updatePackData(packID); err != nil {
			return false, err
		}
	}
	return packChanged, nil
}

func getFileHash(fPath string) (string, error) {
	hash, err := hashSumFile(fPath)
	if err != nil {
		return "", err
	}
	return hash, nil
}
