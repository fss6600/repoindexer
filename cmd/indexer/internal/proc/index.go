package proc

import (
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"log"
	"sort"
)

// dbl - структура данных о файле в пакете в БД
type dbl struct {
	id    int
	path  string
	size  int
	mdate int // todo: change to datetime
}

// Index обработка и индексация пакетов в репозитории
func Index(r *obj.Repo, packs []string) error {
	if len(packs) == 0 {
		// get active packs list
		packs = r.ActivePacks()
	}

	for _, pack := range packs {
		if err := processPackIndex(r, pack); err != nil {
			return err
		}
	}
	return nil
}

type dbt struct {
	id int
	path string
	size int
	mtime int
}

// processPackIndex обрабатывает (индексирует) файлы в указанном пакете
func processPackIndex(r *obj.Repo, pack string) error {



	fmt.Println("start")
	//fsl := []string{}
	fsl := []string{"filesum.s","index.db","contr1.txt","arrange.txt","new.txt","folder//texttospeech.txt"}
	dbl := []dbt{
		{1,"filesum.sum", 123, 321},
		{2,"index.db", 345,5432},
		{3,"arrange-.txt", 345,5432},
	}
	//dbl := []dbt{}

	var (
		fi, di int //  counters
		fPath string // file path on fs
		dbObj dbt // db file object
	)
	fsLastInd := len(fsl) - 1
	dbLastInd := len(dbl) - 1

	fmt.Println("FS:", fsLastInd, "; DB:", dbLastInd)

	sort.Slice(fsl, func(i,j int) bool {return fsl[i] < fsl[j]})
	sort.Slice(dbl, func(i,j int) bool {return dbl[i].path < dbl[j].path})


	for {
		if fi > fsLastInd && di > dbLastInd { // end both lists
			break
		}
		if di > dbLastInd {  // no in DB
			fPath = fsl[fi]
			fmt.Print(fPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to DB")
			fi++  //next path in FS list
			continue
		}
		if fi > fsLastInd {  // not in FS
			dbObj = dbl[di]
			fmt.Println(dbObj.path, ": del from DB")
			di++ // next file obj in DB list
			continue
		}

		fPath = fsl[fi]
		dbObj = dbl[di]

		if fPath == dbObj.path { // in FS, in DB
			fmt.Print(fPath,"=", dbObj.path,)
			_, err := checkSums(fPath)
			if err != nil {
				log.Fatal(err)
			}
			fi++
			di++

		} else if fPath < dbObj.path {  // in FS, not in DB
			fmt.Print(fPath)
			fmt.Print(": calculate sum/date; ")
			fmt.Println(": add to DB")
			fi++
		} else if fPath > dbObj.path {  // not in FS, in DB
			fmt.Println(dbObj.path, ": dell from DB")
			di++
		} else {
			log.Fatal("wrong")
		}
		continue
	}
	return nil
}
