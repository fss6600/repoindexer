package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

<<<<<<< HEAD
func RepoStatus(repo *obj.Repo) {
	fmt.Println("вывод информации о репозитории")
}
=======
func RepoStatus(r *obj.Repo) {
	/*
		todo - сообщения об отсутствии служебных файлов репозитория
		todo - формат даты файлов
>>>>>>> e291739525268f08b2ebe19b20fc56611ccce36a


		todo - общий формат вывода
	*/
	const timeLayout = "2006-01-02 15:04:05"
	fmt.Print("Статус репозитория\t\t")
	SetReglamentMode(r.Path(), "")
	rData := *r.Status()
	fmt.Printf("Пакетов в репозитории \t\t%d\n", rData.TotalCnt)
	fmt.Printf("Проиндексированных пакетов \t%d\n", rData.IndexedCnt)
	fmt.Printf("Заблокированных пакетов \t%d\n", rData.BlockedCnt)
	fmt.Println("---")
	fmt.Printf("index.gz \t%d\t байт от %v\n", rData.IndexSize, rData.IndexMDate.Format(timeLayout))
	fmt.Printf("index.db \t%d\t байт от %v\n", rData.DBSize, rData.DBMDate.Format(timeLayout))
	fmt.Printf("index.gz.sha1 \t%d\t байт от %v\n", rData.HashSize, rData.HashMDate.Format(timeLayout))
	if rData.DBMDate.UnixNano() > rData.IndexMDate.UnixNano() {
		fmt.Println("\nиндекс-файл старше файла БД - произведите выгрузку данных командой 'populate'")
	}
}
