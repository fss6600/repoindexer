package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func ClearDB(r *obj.Repo, cmd string) {
	const tmplErrMsg = "error::cleardb:"
	switch cmd {
	case "index", "all":
		fmt.Print("очистка данных индексации пакетов...")
		err := r.DBCleanPackages()
		utils.CheckError(fmt.Sprintf("\n%v:index:", tmplErrMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "alias":
		fmt.Print("очистка данных псевдонимов...")
		err := r.DBCleanAliases()
		utils.CheckError(fmt.Sprintf("\n%v:alias:", tmplErrMsg), &err)
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "status":
		fmt.Print("очистка данных блокировки...")
		err := r.DBCleanStatus()
		utils.CheckError(fmt.Sprintf("\n%v:status:", tmplErrMsg), &err)
		fmt.Println("OK")
	default:
		panic("укажите одну категорию из списка: index | alias | status | all")
	}
	fmt.Println("\n\tвыгрузите данные в индекс-файл командой 'populate'")
}
