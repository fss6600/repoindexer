package proc

import (
	"encoding/json"
	"path"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

var err error

func Populate(r *obj.Repo) {
	const tmplErrMsg = "error::populate:"
	CheckRegl(r.Path())
	type packages map[string]obj.HashedPackData
	packCh := make(chan obj.HashedPackData)
	packDataList := packages{}

	// список пакетов в из БД
	go func() {
		err := r.HashedPackages(packCh)
		utils.CheckError(tmplErrMsg, &err)
	}() //todo добавить канал ошибок,

	for packData := range packCh { //todo прогон серез select c  обработкой ошибок
		packDataList[packData.Name] = packData // добавляем в map
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), obj.Indexgz)

	// выгрузка данныз из БД в json файл
	err = utils.WriteGzip(jsonData, fp)
	utils.CheckError(tmplErrMsg, &err)

	// подсчет hash суммы индекс-файла
	hash, err := utils.HashSumFile(fp)
	utils.CheckError(tmplErrMsg, &err)

	// запись хэш-суммы индекс-файла
	err = utils.WriteGzipHash(fp, hash)
	utils.CheckError(tmplErrMsg, &err)
}
