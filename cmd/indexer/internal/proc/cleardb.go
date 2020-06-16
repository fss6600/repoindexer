package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func ClearDB(r *obj.Repo, cmd string) {
	const errClrDBMsg = errMsg + ":Cleardb:"
	switch cmd {
	case "index", "all":
		fmt.Print("Очистка данных индексации пакетов...")
		err = r.DBCleanPackages()
		utils.CheckError(fmt.Sprintf("\n%v:index:", errClrDBMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "alias":
		fmt.Print("Очистка данных псевдонимов...")
		err = r.DBCleanAliases()
		utils.CheckError(fmt.Sprintf("\n%v:alias:", errClrDBMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "status":
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
