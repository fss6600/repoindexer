package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

// List выводит на консоль информацию о статусе пакетов в репозитории
func List(r *obj.Repo, cmd string) {
	switch cmd {
	case "all":
		ch := make(chan *obj.ListData)
		fmt.Printf(tmplListOut, "СТАТ", "ПАКЕТ (ПСЕВДОНИМ)")
		fmt.Println("------", "-----------------")
		go r.List(ch)
		for data := range ch {
			switch data.Status {
			case obj.PackStatusBlocked:
				fmt.Printf(tmplListOut, "блок", data.Name)
			case obj.PackStatusActive:
				fmt.Printf(tmplListOut, "", data.Name)
			case obj.PackStatusNotIndexed:
				fmt.Printf(tmplListOut, "!инд", data.Name)
			}
		}
	case "indexed":
		for _, pack := range r.Packages() {
			fmt.Println(pack)
		}
	case "noindexed":
		for _, pack := range r.NoIndexedPacks() {
			fmt.Println(pack)
		}
	case "blocked":
		for _, pack := range r.DisabledPacks() {
			fmt.Println(pack)
		}
	default:
		panic(fmt.Sprintf("неверно указана команда: %v", cmd))
	}
}
