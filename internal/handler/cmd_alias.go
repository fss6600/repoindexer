package handler

import (
	"fmt"
	"strings"
)

// Alias обрабатывает команду alias
// без параметров выводит список установленных псевдонимов
// set - устанавливает псевдоним для пакета принимает параметр
// (или параметры через пробел) вида ПАКЕТ=ПСЕВДОНИМ
// del - удаляет псевдоним пакета принимает параметр
// (или параметры через пробел) с именем пакета или наименованием псевдонима
func Alias(r *Repo, cmd string, aliases []string) error {
	switch cmd {
	case "set":
		for _, alias := range aliases {
			alias := strings.Split(alias, "=")
			if len(alias) != 2 {
				return &InternalError{
					Text:   fmt.Sprintf("неверный псевдоним - %q\n\n\tформат: alias set ПАКЕТ=ПСЕВДОНИМ", alias[0]),
					Caller: "Alias",
				}
			}
			if err = r.setAlias(alias); err != nil {
				return err
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
				return &InternalError{
					Text:   fmt.Sprintf("неверный псевдоним - %q\n\n\tформат: alias del [ПАКЕТ=]ПСЕВДОНИМ", alias[0]),
					Caller: "Alias",
				}
			}
			if err = r.delAlias(alias); err != nil {
				return err
			}
		}
		fmt.Println(doPopMsg)
	case "", "show":
		// show alias info
		aliases := r.aliases()
		if len(aliases) == 0 {
			fmt.Println("Список псевдонимов пуст")
		} else {
			for _, aliasPair := range aliases {
				fmt.Printf("%v=%v\n", aliasPair[0], aliasPair[1])
			}
		}
	default:
		return &InternalError{
			Text:   fmt.Sprintf("неверная команда %q. укажите одну из [ 'set' | 'del' | 'show' ]", cmd),
			Caller: "Alias",
		}
	}
	return nil
}
