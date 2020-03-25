package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func RepoStatus(r *obj.Repo) {
	/*
		todo - сообщения об отсутствии служебных файлов репозитория
		todo - формат даты файлов
	*/
	const tmplErrMsg = "error::status:"
	const timeLayout = "2006-01-02 15:04:05"
	rData, err := r.Status()
	utils.CheckError(tmplErrMsg, &err)

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

	if unIndexed > 0 || unIndexed < 0 {
		fmt.Println("\n\tтребуется индексация репозитория")
	} else if rData.IndexMDate.IsZero() {
		fmt.Println("\n\tиндекс-файл отсутствует - произведите выгрузку данных командой 'populate'")
	} else if rData.DBMDate.UnixNano() > rData.IndexMDate.UnixNano() {
		fmt.Println("\n\tиндекс-файл старше файла БД - произведите выгрузку данных командой 'populate'")
	}
}
