package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func RepoStatus(r *obj.Repo) {
	const errRStatMsg = errMsg + ":status:"
	rData, err := r.Status()
	utils.CheckError(errRStatMsg, &err)

	var reglStatus string
	switch ReglIsSet(r.Path()) {
	case true:
		reglStatus = "on"
	default:
		reglStatus = "off"
	}
	unIndexed := rData.TotalCnt - (rData.IndexedCnt + rData.BlockedCnt)
	tmpl := "%-40s%v\n"

	fmt.Printf(tmpl, "Статус регламента", reglStatus)
	fmt.Println("")
	fmt.Printf(tmpl, "Пакетов в репозитории", rData.TotalCnt)
	fmt.Printf(tmpl, "Пакетов проиндексировано", rData.IndexedCnt)
	fmt.Printf(tmpl, "Пакетов заблокировано", rData.BlockedCnt)
	if unIndexed > 0 {
		fmt.Printf(tmpl, "Пакетов непроиндексировано", unIndexed)
	} else if unIndexed < 0 {
		fmt.Printf(tmpl, "Удалено пакетов из репозитория", unIndexed*-1)
	}
	fmt.Println("")
	if rData.DBSize > -1 {
		fmt.Printf("index.db \t%d\t байт от %v\n", rData.DBSize, rData.DBMDate.Format(timeLayout))
	} else {
		fmt.Println("index.db \t\t нет данных")
	}
	if rData.IndexSize > -1 {
		fmt.Printf("index.gz \t%d\t байт от %v\n", rData.IndexSize, rData.IndexMDate.Format(timeLayout))
	} else {
		fmt.Println("index.gz \t\t нет данных")
	}
	if rData.HashSize > -1 {
		fmt.Printf("index.gz.sha1 \t%d\t байт от %v\n", rData.HashSize, rData.HashMDate.Format(timeLayout))
	} else {
		fmt.Println("index.gz.sha1 \t\t нет данных")
	}
	fmt.Println("")

	vMaj, vMin, err := r.VersionDB()
	if err != nil {
		panic("\n\tНе удалось получить версию БД. Требуется переиндексация репозитория")
	}

	tmpl = "Версия БД %-40s%d.%d\n"
	fmt.Printf(tmpl, "программы: ", obj.DBVersionMajor, obj.DBVersionMinor)
	fmt.Printf(tmpl, "репозитория: ", vMaj, vMin)

	err = r.CheckDBVersion()
	utils.CheckError("", &err)

	obj.ShowEmptyExecFiles(r) // проверка на пустые исполняемый файлы

	if unIndexed > 0 || unIndexed < 0 {
		fmt.Println(doIndexMsg)
	} else if rData.IndexMDate.IsZero() {
		fmt.Print("\n\tИндекс-файл отсутствует")
		fmt.Println(doPopMsg)
	} else if rData.DBMDate.UnixNano() > rData.IndexMDate.UnixNano() {
		fmt.Print("\n\tИндекс-файл старше файла БД")
		fmt.Println(doPopMsg)
	}
}
