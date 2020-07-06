package handler

import (
	"fmt"
)

// ExecFile обрабатывает команду `exec` поиск и установка исполняемого файла
// для пакета или всех пакетов в репозитории
// check - проверка данных об исполняемом файле пакета (пакетов), при отсутствии установка
// set - установка данных об исполняемом файле
// del - удаление данных об исполняемом файле
// show - вывод информации об установленных исполняемых файлах пакета
func ExecFile(r *Repo, cmd string, packs []string) error {
	packsCount := len(packs)

	var force bool
	if cmd == "set" {
		force = true
	}

	switch cmd {
	case "check", "set":
		if packsCount == 0 && cmd == "set" {
			if !userAccept("Обработать данные об исполняемом файле во всех пакетах?") {
				return nil
			}
		}
		if packsCount == 0 {
			packs = r.ActivePacks()
		}
		fmt.Print("Проверка (установка) исполняемого файла для пакета:\n\n")
		for _, pack := range packs {
			if err = r.execFileSet(pack, force); err != nil {
				return err
			}
		}
	case "del":
		if packsCount == 0 {
			if !userAccept("Удалить данные об исполняемом файле во всех пакетах?") {
				return nil
			}
			packs = r.ActivePacks()
		}
		for _, pack := range packs {
			if err = r.execFileDel(pack); err != nil {
				return err
			}
		}
	case "show":
		if packsCount == 0 {
			packs = r.ActivePacks()
		}

		for _, pack := range packs {
			execFile, err := r.execFileInfo(pack)
			if err != nil {
				return err
			}
			fmt.Printf("\t%v: %v\n", pack, execFile)
		}
	default:
		return &InternalError{
			Text:   fmt.Sprintf("неверная команда '%v'. укажите одну из [ 'check' | 'set' | 'del' | 'show' ]", cmd),
			Caller: "ExecFile",
		}
	}
	return nil
}
