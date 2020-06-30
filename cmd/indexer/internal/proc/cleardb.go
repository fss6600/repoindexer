package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

// ClearDB обрабатывает команду cleardb
// all - удаляет все данные из БД
// index - удаляет данные об индексации из БД
// alias - удаляет данные о псевдонимах из БД
// status - удаляет данные о блокировках из БД
func ClearDB(r *obj.Repo, cmd string) {
	const errClrDBMsg = errMsg + ":Cleardb:"
	var msg string
	if cmd == "all" {
		msg = "Удаляются все данные репозитория из БД"
	} else {
		msg = "Удаляются данные индекса из БД"
	}
	switch cmd {
	case "index", "all":
		if !utils.UserAccept(msg) {
			return
		}
		fmt.Print("Очистка данных индексации пакетов...")
		err = r.DBCleanPackages()
		utils.CheckError(fmt.Sprintf("\n%v:index:", errClrDBMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "alias":
		if cmd != "all" && !utils.UserAccept("Удаляются данные псевдонимов из БД") {
			return
		}
		fmt.Print("Очистка данных псевдонимов...")
		err = r.DBCleanAliases()
		utils.CheckError(fmt.Sprintf("\n%v:alias:", errClrDBMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "status":
		if cmd != "all" && !utils.UserAccept("Удаляются данные блокирововк из БД") {
			return
		}
		fmt.Print("Очистка данных блокировки...")
		err = r.DBCleanStatus()
		utils.CheckError(fmt.Sprintf("\n%v:status:", errClrDBMsg), &err)
		fmt.Println("OK")
	default:
		panic("укажите одну категорию из списка: index | alias | status | all")
	}
	if cmd == "alias" {
		fmt.Println(doPopMsg)
	} else {
		fmt.Println(doIndexMsg)
	}
}
