package handler

import (
	"fmt"
)

// List выводит на консоль информацию о статусе пакетов в репозитории
func List(r *Repo, cmd string) error {
	const tmplListOut = "[%4v] %v\n"
	switch cmd {
	case "all":
		ch := make(chan *ListData)
		fmt.Printf(tmplListOut, "СТАТ", "ПАКЕТ (ПСЕВДОНИМ)")
		fmt.Println("------", "-----------------")
		go r.List(ch)
		for data := range ch {
			switch data.Status {
			case PackStatusBlocked:
				fmt.Printf(tmplListOut, "блок", data.Name)
			case PackStatusActive:
				fmt.Printf(tmplListOut, "", data.Name)
			case PackStatusNotIndexed:
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
		return &InternalError{
			Text:   fmt.Sprintf("неверно указана команда: %q", cmd),
			Caller: "List",
		}
	}
	return nil
}
