package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func MigrateDB(r *obj.Repo) {
	const ErrMirateMsg = errMsg + ":Migrate:"
	fmt.Println("Миграция ДБ:")
	vMaj, vMin, err := r.VersionDB()
	utils.CheckError(ErrMirateMsg, &err)
	if obj.DBVersionMajor > vMaj {
		panic("\n\tТребуется переиндексация репозитория")
	} else if obj.DBVersionMajor < vMaj || obj.DBVersionMinor < vMin {
		panic("Возможно вы используете старую версию программы")
	} else if obj.DBVersionMajor == vMaj && obj.DBVersionMinor == vMin {
		panic("Миграция не требуется")
	}
	// подготовка списка заблокированных пакетов
	//blocked := r.DisabledPacks()
	// подготовка списка псевдонимов
	//aliases := r.Aliases()

	// переименование файла БД и индекс-файлов

	// инициализация репозитория

	// ввод псевдонимов в БД

	// ввод заблокированных в БД

	// при удачном раскладе - возврат файла прежней БД и индекс-файлов
}
