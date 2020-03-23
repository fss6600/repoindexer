package proc

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

var err error

func Populate(r *obj.Repo) error {
	CheckRegl(r.Path())
	type packages map[string]obj.HashedPackData
	packCh := make(chan obj.HashedPackData)
	packDataList := packages{}

	// список пакетов в из БД
	go r.HashedPackages(packCh) //todo добавить канал ошибок,

	for packData := range packCh { //todo прогон серез select c  обработкой ошибок
		packDataList[packData.Name] = packData // добавляем в map
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), obj.Indexgz)

	// выгрузка данныз из БД в json файл
	err = writeGzip(jsonData, fp)
	utils.CheckError("populate", &err)

	// подсчет hash суммы индекс-файла
	hash, err := utils.HashSumFile(fp)
	utils.CheckError("populate", &err)

	// запись хэш-суммы индекс-файла
	err = writeGzipHash(fp, hash)
	utils.CheckError("populate", &err)

	return nil // todo убрать возврат ошибки; - через панику
}

func writeGzip(jsonData []byte, fp string) error {
	errMsg := fmt.Errorf(":writeGzip: %v", err)
	indexFile, err := os.Create(fp)
	if err != nil {
		return errMsg
	}
	zw := gzip.NewWriter(indexFile)
	defer func() {
		_ = zw.Close()
		_ = indexFile.Close()
	}()

	_, err = zw.Write(jsonData)
	if err != nil {
		return errMsg
	}
	return nil
}

func writeGzipHash(fp, hash string) error {
	errMsg := fmt.Errorf(":writeGzipHash: %v", err)
	indexFileHash, err := os.Create(fp + ".sha1")
	if err != nil {
		return errMsg
	}
	defer indexFileHash.Close()

	_, err = indexFileHash.Write([]byte(hash))
	if err != nil {
		return errMsg
	}
	return nil
}
