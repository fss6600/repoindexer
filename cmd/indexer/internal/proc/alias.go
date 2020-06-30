package proc

import (
	"fmt"
	"strings"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

// Alias обрабатывает команду alias
// без параметров выводит список установленных псевдонимов
// set - устанавливает псевдоним для пакета принимает параметр
// (или параметры через пробел) вида ПАКЕТ=ПСЕВДОНИМ
// del - удаляет псевдоним пакета принимает параметр
// (или параметры через пробел) с именем пакета или наименованием псевдонима
func Alias(r *obj.Repo, cmd string, aliases []string) {
	const errAliasMsg = errMsg + ":Alias:"

	switch cmd {
	case "set":
		for _, alias := range aliases {
			alias := strings.Split(alias, "=")
			if len(alias) != 2 {
				throw(alias[0])
			}
			if err = r.SetAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					panic(fmt.Sprintf("%v\n", er))
				} else {
					panic(fmt.Errorf("%v:%v:%v", errAliasMsg, "set", err))
				}
			}
		}
		fmt.Println(doPopMsg)
	case "del":
		for _, alias := range aliases {
			als := strings.Split(alias, "=")
			if l := len(als); l == 1 {
				alias = als[0]
			} else if l == 2 {
				alias = als[1]
			} else {
				throw(alias)
			}
			if err = r.DelAlias(alias); err != nil {
				if er, ok := err.(obj.ErrAlias); ok {
					panic(fmt.Sprintf("%v\n", er))
				} else {
					panic(fmt.Errorf("%v:%v%v", errAliasMsg, "del", err))
				}
			}
		}
		fmt.Println(doPopMsg)
	case "", "show":
		// show alias info
		aliases := r.Aliases()
		if len(aliases) == 0 {
			fmt.Println("Список псевдонимов пуст")
		} else {
			for _, aliasPair := range aliases {
				fmt.Printf("%v=%v\n", aliasPair[0], aliasPair[1])
			}
		}
	default:
		panic(fmt.Sprintf("неверная соманда '%v'. укажите одну из [ 'set' | 'del' | 'show' ]\n", cmd))
	}
}

func throw(alias string) {
	panic(fmt.Sprintf("неверно задан псевдоним - (%v)\n\n\t"+
		"формат: alias set ПАКЕТ=ПСЕВДОНИМ", alias))
}
