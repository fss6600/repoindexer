package proc

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func Populate(r *obj.Repo) {
	err = r.CheckDBVersion()
	utils.CheckError("", &err)
	err = r.CheckEmptyExecFiles()
	utils.CheckError("", &err)

	const errPopMsg = errMsg + ":populate:"
	fmt.Print("Выгрузка данных в индекс файл: ")
	CheckRegl(r.Path())
	type packages map[string]obj.HashedPackData
	packCh := make(chan obj.HashedPackData)
	packDataList := packages{}

	// список пакетов в из БД
	go func() {
		err = r.HashedPackages(packCh)
		utils.CheckError(errPopMsg, &err)
	}() //todo добавить канал ошибок,

	for packData := range packCh { //todo прогон серез select c  обработкой ошибок
		packDataList[packData.Name] = packData // добавляем в map
	}

	if len(packDataList) == 0 {
		panic("нет данных - требуется индексация репозитория")
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), obj.IndexGZ)

	// выгрузка данныз из БД в json файл
	err = utils.WriteGzip(jsonData, fp)
	utils.CheckError(errPopMsg, &err)

	// подсчет hash суммы индекс-файла
	hash, err := utils.HashSumFile(fp)
	utils.CheckError(errPopMsg, &err)

	// запись хэш-суммы индекс-файла
	err = utils.WriteGzipHash(fp, hash)
	utils.CheckError(errPopMsg, &err)
	fmt.Println("OK")
}
