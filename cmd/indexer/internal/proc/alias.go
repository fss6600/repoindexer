package proc

import (
	"fmt"
	"strings"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

func Alias(r *obj.Repo, cmd string, aliases []string) {
	const tmplErrMsg = "error::alias:"
	switch cmd {
	case "set":
		// set aliases
		if len(aliases) == 0 {
			panic("укажите по крайней мере 1 пару 'ПСЕВДОНИМ'='ПАКЕТ'")
		}
		for _, alias := range aliases {
			alias := strings.Split(alias, "=")
			if len(alias) != 2 {
				panic(fmt.Sprintf("неверно задан псевдоним - (%v)\n\n\t"+
					"формат: alias set 'ПСЕВДОНИМ'='ПАКЕТ'", alias))
			}
			if err := r.SetAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					panic(fmt.Sprintf("%v\n", er))
				} else {
					panic(fmt.Errorf("%v:%v:%v", tmplErrMsg, "set", err))
				}
			}
			fmt.Printf("установлен псевдоним: [ %v ]=[ %v ]\n", alias[0], alias[1])
		}
		fmt.Println("\n\tвыгрузите данные командой 'populate'")
	case "del":
		// del aliases
		if len(aliases) == 0 {
			panic("укажите по крайней мере 1 псевдоним")
		}
		for _, alias := range aliases {
			if err := r.DelAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					panic(fmt.Sprintf("%v\n", er))
				} else {
					panic(fmt.Errorf("%v:%v%v", tmplErrMsg, "del", err))
				}
			}
			fmt.Printf("удален псевдоним: [ %v ]\n", alias)
		}
		fmt.Println("\n\tвыгрузите данные командой 'populate'")
	case "", "show":
		// show alias info
		aliases := r.Aliases()
		if len(aliases) == 0 {
			fmt.Println("список псевдонимов пуст")
		} else {
			for _, aliasPair := range aliases {
				fmt.Printf("[ %v ]=[ %v ]\n", aliasPair[0], aliasPair[1])
			}
		}
	default:
		panic(fmt.Sprintf("неверная соманда '%v'. укажите одну из [ 'set' | 'del' | 'show' ]\n", cmd))
	}
}
