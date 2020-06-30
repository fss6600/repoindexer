package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

// TODO: алгоритм миграции с сохранением данных БД

// MigrateDB обрабатывает команду `migrate`
// сохраняет данные псевдонимов и блокировок пакетов при миграции
// и импортирует обратно после удаления данных из БД
// Миграция требуется при изменении структуры БД
func MigrateDB(r *obj.Repo) {
	checkRegl(r.Path())
	if !utils.UserAccept("\nДанная операция заменит файлы БД и индекса." +
		"\nУбедитесь, что у Вас есть резервная копия") {
		return
	}
	const ErrMigrateMsg = errMsg + ":Migrate:"
	fmt.Println("")
	vMaj, vMin, err := r.VersionDB()
	utils.CheckError(ErrMigrateMsg, &err)
	if obj.DBVersionMajor > vMaj {
		panic("\n\tТребуется переиндексация репозитория")
	} else if obj.DBVersionMajor < vMaj || obj.DBVersionMinor < vMin {
		panic("Возможно вы используете старую версию программы")
	} else if obj.DBVersionMajor == vMaj && obj.DBVersionMinor == vMin {
		panic("Миграция не требуется")
	}
	tmpl := "%-30s: "
	//подготовка списка заблокированных пакетов
	fmt.Printf(tmpl, "сохранение alias, blocked")
	blocked := r.DisabledPacks()
	//подготовка списка псевдонимов
	aliases := r.Aliases()
	fmt.Println("OK")

	// close DB
	fmt.Printf(tmpl, "Закрытие БД")
	err = r.Close()
	utils.CheckError(ErrMigrateMsg, &err)
	fmt.Println("OK")

	// очистка старых файлов БД и индекса
	fmt.Printf(tmpl, "удаление файлов БД и индекса")
	err = obj.CleanForMigrate(r)
	utils.CheckError(ErrMigrateMsg, &err)
	fmt.Println("OK")

	// инициализация репозитория
	fmt.Printf(tmpl, "инициализация репозитория")
	err = obj.InitDB(r.Path())
	utils.CheckError(ErrMigrateMsg, &err)

	// connect to DB
	err = r.OpenDB()
	utils.CheckError(ErrMigrateMsg, &err)

	// ввод псевдонимов в БД
	fmt.Println("восстановление alias, blocked")
	for _, alias := range aliases {
		err = r.SetAlias(alias)
		utils.CheckError(ErrMigrateMsg, &err)
	}

	// ввод заблокированных в БД
	for _, pack := range blocked {
		err = r.DisablePack(pack)
		utils.CheckError(ErrMigrateMsg, &err)
	}
	fmt.Println("Миграция завершена")
	fmt.Println("\n\tТребуется индексация репозитория")
}
