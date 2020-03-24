package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

// ...
func List(r *obj.Repo, cmd string) {
	switch cmd {
	case "all":
		ch := make(chan *obj.ListData)
		line := fmt.Sprintln("[%4v] %v")
		fmt.Printf(line, "СТАТ", "ПАКЕТ (ПСЕВДОНИМ)")
		fmt.Println("------", "-----------------")
		go r.List(ch)
		for data := range ch {
			switch data.Status {
			case 0:
				fmt.Printf(line, "блок", data.Name)
			case 1:
				fmt.Printf(line, "", data.Name)
			case -1:
				fmt.Printf(line, "!инд", data.Name)
			}
		}
	case "indexed":
		for _, pack := range r.IndexedPacks() {
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
