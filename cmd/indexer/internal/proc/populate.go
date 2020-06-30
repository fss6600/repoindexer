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

type packages map[string]obj.HashedPackData

const errPopMsg = errMsg + ":populate:"

// Populate выгружает данные об индексации репозитория в индекс-файл
func Populate(r *obj.Repo) {
	err = r.CheckDBVersion()
	utils.CheckError("", &err)
	err = r.CheckEmptyExecFiles()
	utils.CheckError("", &err)

	fmt.Print("Выгрузка данных в индекс файл: ")
	checkRegl(r.Path())

	// список пакетов в из БД
	packCh := make(chan obj.HashedPackData)
	go func() {
		if err = r.HashedPackages(packCh); err != nil {
			fmt.Println(errPopMsg, err)
			os.Exit(1)
		}
	}()

	packDataList := packages{}
	for packData := range packCh {
		packDataList[packData.Name] = packData // добавляем в map
	}

	if len(packDataList) == 0 {
		panic("нет данных - требуется индексация репозитория")
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), obj.IndexGZ)

	// выгрузка данных из БД в json файл
	err = writeGzip(jsonData, fp)
	utils.CheckError(errPopMsg, &err)

	// подсчет hash суммы индекс-файла
	hash, err := utils.HashSumFile(fp)
	utils.CheckError(errPopMsg, &err)

	// запись хэш-суммы индекс-файла
	err = writeGzipHash(fp, hash)
	utils.CheckError(errPopMsg, &err)
	fmt.Println("OK")
}

func writeGzipHash(fp, hash string) error {
	errMsg := "writeGzipHash: %v"
	indexFileHash, err := os.Create(fp + ".sha1")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	defer func() {
		err := indexFileHash.Close()
		utils.CheckError("writeGzipHash", &err)
	}()

	_, err = indexFileHash.Write([]byte(hash))
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func writeGzip(jsonData []byte, fp string) error {
	errMsg := ":WriteGzip: %v"
	indexFile, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	zw := gzip.NewWriter(indexFile)
	defer func() {
		err = zw.Close()
		utils.CheckError("writeGzip", &err)
		err = indexFile.Close()
		utils.CheckError("writeGzip", &err)
	}()

	_, err = zw.Write(jsonData)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}
