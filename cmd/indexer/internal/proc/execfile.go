package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const errExecFileMsg = ":ExecFile:"

// Поиск и установка исполняемого файла для пакета
func ExecFile(r *obj.Repo, cmd string, packs []string) {
	packsCount := len(packs)

	var force bool
	if cmd == "set" {
		force = true
	}

	switch cmd {
	case "check", "set":
		if packsCount == 0 && cmd == "set" {
			switch utils.UserAccept("Обработать данные об исполняемом файле во всех пакетах?") {
			case false:
				return
			}
		}
		if packsCount == 0 {
			packs = r.ActivePacks()
		}
		fmt.Print("Проверка (установка) исполняемого файла для пакета:\n\n")
		for _, pack := range packs {

			err = r.ExecFileSet(pack, force)
			utils.CheckError(errExecFileMsg, &err)
		}
	case "del":
		if packsCount == 0 {
			switch utils.UserAccept("Удалить данные об исполняемом файле во всех пакетах?") {
			case true:
				packs = r.ActivePacks()
			case false:
				return
			}
		}
		for _, pack := range packs {
			err = r.ExecFileDel(pack)
			utils.CheckError(errExecFileMsg, &err)
		}
	case "show":
		if packsCount == 0 {
			packs = r.ActivePacks()
		}

		for _, pack := range packs {
			execFile, err := r.ExecFileInfo(pack)
			utils.CheckError(errExecFileMsg, &err)
			fmt.Printf("\t%v: %v\n", pack, execFile)
		}
	default:
		panic(fmt.Sprintf("неверная соманда '%v'. укажите одну из [ 'check' | 'set' | 'del' | 'show' ]\n", cmd))
	}
}
