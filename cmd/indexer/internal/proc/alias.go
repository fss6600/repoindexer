package proc

import (
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"strings"
)

func Alias(r *obj.Repo, cmd string, aliases []string) {
	switch cmd {
	case "set":
		// set aliases
		if len(aliases) == 0 {
			fmt.Println("укажите по крайней мере 1 пару 'ПСЕВДОНИМ'='ПАКЕТ'")
			return
		}
		for _, alias := range aliases {
			alias := strings.Split(alias, "=")
			if len(alias) != 2 {
				fmt.Printf("неверно задан псевдоним - [ %v ]\n", alias)
				fmt.Println("формат: alias set 'ПСЕВДОНИМ'='ПАКЕТ'")
			}
			if err := r.SetAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					fmt.Printf("%v\n", er)
					return
				} else {
					panic(fmt.Errorf(":aliases::set:%v", err))
				}
			}
			fmt.Printf("установлен псевдоним: [ %v ]=[ %v ]\n", alias[0], alias[1])
		}
		fmt.Println("выгрузите данные командой 'populate'")
	case "del":
		// del aliases
		if len(aliases) == 0 {
			fmt.Println("укажите по крайней мере 1 псевдоним")
			return
		}
		for _, alias := range aliases {
			if err := r.DelAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					fmt.Printf("%v\n", er)
					return
				} else {
					panic(fmt.Errorf(":aliases::del:%v", err))
				}
			}
			fmt.Printf("удален псевдоним: [ %v ]\n", alias)
		}
		fmt.Println("выгрузите данные командой 'populate'")
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
		fmt.Printf("неверная соманда '%v'. укажите одну из [ 'set' | 'del' | 'show' ]\n", cmd)
	}
}
