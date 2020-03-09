package proc

import (
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"log"
	"sort"
)

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, packs []string) error {
	r.PrepareDisabledPaksList() //

	if len(packs) == 0 {
		// get active packs list
		packs = r.ActivePacks()
	}
	fmt.Println(packs)
	for _, pack := range packs {
		fmt.Println("[", pack, "]")
		if err := processPackIndex(r, pack); err != nil {
			return err
		}
	}
	return nil
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) error {
	//todo: add check pack in disabled list - clean db then
	//if pack == "" {
	//	return errors.New("пустое имя пакета")
	//}
	fsl, err := r.FilesPackRepo(pack)
	if err != nil {
		return err
	}
	dbl, err := r.FilesPackDB(pack)
	if err != nil {
		return err
	}

	var (
		fi, di int          //  counters
		fsPath string       // file path on fs
		dbData obj.FileInfo // db file object
		changed bool // package has changes
	)
	fsLastInd := len(fsl) - 1
	dbLastInd := len(dbl) - 1

	//fmt.Println("FS:", fsLastInd, "; DB:", dbLastInd)

	sort.Slice(fsl, func(i,j int) bool {return fsl[i] < fsl[j]})
	sort.Slice(dbl, func(i,j int) bool {return dbl[i].Path < dbl[j].Path})


	for {
		if fi > fsLastInd && di > dbLastInd { // end both lists
			break
		}
		if di > dbLastInd {  // no in DB
			fsPath = fsl[fi]
			fmt.Print(fsPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to DB")
			fi++  //next path in FS list
			if !changed {changed = true}
			continue
		}
		if fi > fsLastInd {  // not in FS
			dbData = dbl[di]
			fmt.Println(dbData.Path, ": del from DB")
			di++ // next file obj in DB list
			continue
		}

		fsPath = fsl[fi]
		dbData = dbl[di]

		if fsPath == dbData.Path { // in FS, in DB
			fmt.Print(fsPath,"=", dbData.Path,)
			_, err := internal.CheckSums(fsPath)
			if err != nil {
				log.Fatal(err)
			}
			fi++
			di++

		} else if fsPath < dbData.Path { // in FS, not in DB
			fmt.Print(fsPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to DB")
			if !changed {changed = true}
			fi++
		} else if fsPath > dbData.Path { // not in FS, in DB
			fmt.Println(dbData.Path, ": dell from DB")
			di++
		} else {
			log.Fatal("wrong")
		}
		continue
	}
	return nil
}
