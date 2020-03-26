package proc

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

func Populate(r *obj.Repo) {
	const errMsg = errMsg + ":populate:"
	fmt.Print("Выгрузка данных в индекс файл: ")
	CheckRegl(r.Path())
	type packages map[string]obj.HashedPackData
	packCh := make(chan obj.HashedPackData)
	packDataList := packages{}

	// список пакетов в из БД
	go func() {
		err = r.HashedPackages(packCh)
		utils.CheckError(errMsg, &err)
	}() //todo добавить канал ошибок,

	for packData := range packCh { //todo прогон серез select c  обработкой ошибок
		packDataList[packData.Name] = packData // добавляем в map
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), obj.Indexgz)

	// выгрузка данныз из БД в json файл
	err = utils.WriteGzip(jsonData, fp)
	utils.CheckError(errMsg, &err)

	// подсчет hash суммы индекс-файла
	hash, err := utils.HashSumFile(fp)
	utils.CheckError(errMsg, &err)

	// запись хэш-суммы индекс-файла
	err = utils.WriteGzipHash(fp, hash)
	utils.CheckError(errMsg, &err)
	fmt.Println("OK")
}
