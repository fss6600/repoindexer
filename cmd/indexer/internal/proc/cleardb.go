package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

func ClearDB(r *obj.Repo, cmd string) {
	switch cmd {
	case "index", "all":
		fmt.Print("очистка данных индексации пакетов...")
		err := r.DBCleanPackages()
		if err != nil {
			fmt.Println()
			panic(fmt.Sprintf(":cleardb::index:%v", err))
		}
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "alias":
		fmt.Print("очистка данных псевдонимов...")
		err := r.DBCleanAliases()
		if err != nil {
			fmt.Println()
			panic(fmt.Sprintf(":cleardb::alias:%v", err))
		}
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "status":
		fmt.Print("очистка данных блокировки...")
		err := r.DBCleanStatus()
		if err != nil {
			fmt.Println()
			panic(fmt.Sprintf(":cleardb::status:%v", err))
		}
		fmt.Println("OK")
	default:
		panic("укажите одну категорию из списка: index | alias | status | all")
	}
	fmt.Println("\n\tвыгрузите данные в индекс-файл командой 'populate'")
}
