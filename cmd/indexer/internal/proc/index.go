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
	//r.PrepareDisabledPaksList() //

	//asd  := len([]string{}) == 0
	//fmt.Println(asd)

	if len(packs) == 0 {
		// получает актуальные активные пакеты в репозитории
		packs = r.ActivePacks()
	} else {
		// проверяем на наличие указанных пакетов в репозитории
		for _, pack := range packs {
			if err := r.CheckExists(pack); err != nil {
				return err
			}
		}
	}
	fmt.Println(packs)
	for _, pack := range packs {
		if pack == "" {
			return errors.New("error: задано пустое имя пакета")
		}
		fmt.Println("[", pack, "]")
		if err := processPackIndex(r, pack); err != nil {
			return err
		}
	}
	//todo: add check pack in disabled list - clean db then
	r.Clean()
	return nil
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) error {
	var (
		fsl     []string       // список файлов пакета в репозитории
		dbl     []obj.FileInfo // список файлов пакета в БД
		err     error
		fi, di  int          //  counters
		fsPath  string       // file path on fs
		dbData  obj.FileInfo // db file object
		changed bool         // package has changes
	)

	fsl, err = r.FilesPackRepo(pack)
	if err != nil {
		return err
	}
	dbl, err = r.FilesPackDB(pack)
	if err != nil {
		return err
	}

	fsLastInd := len(fsl) - 1
	dbLastInd := len(dbl) - 1

	//fmt.Println("FS:", fsLastInd, "; db:", dbLastInd)

	sort.Slice(fsl, func(i, j int) bool { return fsl[i] < fsl[j] })
	sort.Slice(dbl, func(i, j int) bool { return dbl[i].Path < dbl[j].Path })

	for {
		if fi > fsLastInd && di > dbLastInd { // end both lists
			break
		}
		if di > dbLastInd { // no in db
			fsPath = fsl[fi]
			fmt.Print(fsPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to db")
			fi++ //next path in FS list
			if !changed {
				changed = true
			}
			continue
		}
		if fi > fsLastInd { // not in FS
			dbData = dbl[di]
			fmt.Println(dbData.Path, ": del from db")
			di++ // next file obj in db list
			continue
		}

		fsPath = fsl[fi]
		dbData = dbl[di]

		if fsPath == dbData.Path { // in FS, in db
			fmt.Print(fsPath, "=", dbData.Path)
			_, err := internal.CheckSums(fsPath)
			if err != nil {
				log.Fatal(err)
			}
			fi++
			di++

		} else if fsPath < dbData.Path { // in FS, not in db
			fmt.Print(fsPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to db")
			if !changed {
				changed = true
			}
			fi++
		} else if fsPath > dbData.Path { // not in FS, in db
			fmt.Println(dbData.Path, ": dell from db")
			di++
		} else {
			log.Fatal("wrong")
		}
		continue
	}
	return nil
}
