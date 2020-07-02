package handler

import (
	"fmt"
)

// RepoStatus обрабатывает команду `status`
// выводит актуальную информацию о репозитории
func RepoStatus(r *Repo) error {
	const timeLayout = "2006-01-02 15:04:05"
	rData, err := r.Status()
	if err != nil {
		return err
	}

	var reglStatus string
	switch reglIsSet(r.path) {
	case true:
		reglStatus = "on"
	default:
		reglStatus = "off"
	}

	unIndexed := rData.TotalCnt - (rData.IndexedCnt + rData.BlockedCnt)
	template := "%-40s%v\n"

	fmt.Printf(template, "Статус регламента", reglStatus)
	fmt.Println()
	fmt.Printf(template, "Пакетов в репозитории", rData.TotalCnt)
	fmt.Printf(template, "Пакетов проиндексировано", rData.IndexedCnt)
	fmt.Printf(template, "Пакетов заблокировано", rData.BlockedCnt)
	if unIndexed > 0 {
		fmt.Printf(template, "Пакетов непроиндексировано", unIndexed)
	} else if unIndexed < 0 {
		fmt.Printf(template, "Удалено пакетов из репозитория", unIndexed*-1)
	}
	fmt.Println()
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
	fmt.Println()

	vMaj, vMin, err := r.VersionDB()
	if err != nil {
		return err
	}

	template = "Версия БД %-40s%d.%d\n"
	fmt.Printf(template, "программы: ", DBVersionMajor, DBVersionMinor)
	fmt.Printf(template, "репозитория: ", vMaj, vMin)

	if err = r.CheckDBVersion(); err != nil {
		return err
	}

	ShowEmptyExecFiles(r) // проверка на пустые исполняемый файлы

	if unIndexed > 0 || unIndexed < 0 {
		fmt.Println(doIndexMsg)
	} else if rData.IndexMDate.IsZero() {
		fmt.Print("\n\tИндекс-файл отсутствует")
		fmt.Println(doPopMsg)
	} else if rData.DBMDate.UnixNano() > rData.IndexMDate.UnixNano() {
		fmt.Print("\n\tИндекс-файл старше файла БД")
		fmt.Println(doPopMsg)
	}
	return nil
}
