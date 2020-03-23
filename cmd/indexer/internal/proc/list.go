package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

func List(r *obj.Repo, cmd string) {
	switch cmd {
	case "all":
		line := fmt.Sprintln("[%4v] %v")
		ch := make(chan *obj.ListData)
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

	case "active":
		for _, pack := range r.ActivePacks() {
			fmt.Println(pack)
		}
	case "disabled":
		for _, pack := range r.DisabledPacks() {
			fmt.Println(pack)
		}
	default:
		panic(fmt.Sprintf("неверно указана команда: %v", cmd))
	}
}
