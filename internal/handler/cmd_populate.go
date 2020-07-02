package handler

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
)

type packages map[string]HashedPackData

// Populate выгружает данные об индексации репозитория в индекс-файл
func Populate(r *Repo) error {
	if err = r.CheckDBVersion(); err != nil {
		return err
	}
	if err = r.CheckEmptyExecFiles(); err != nil {
		return err
	}

	fmt.Print("Выгрузка данных в индекс файл: ")

	if err = checkRegl(r.path); err != nil {
		return err
	}

	// список пакетов в из БД
	packCh := make(chan HashedPackData)
	go func() {
		if err = r.HashedPackages(packCh); err != nil {
			log.Fatal(err)
		}
	}()

	packDataList := packages{}
	for packData := range packCh {
		packDataList[packData.Name] = packData // добавляем в map
	}

	if len(packDataList) == 0 {
		return &internalError{
			Text:   "нет данных - требуется индексация репозитория",
			Caller: "Populate",
		}
	}

	var jsonData []byte
	jsonData, _ = json.MarshalIndent(packDataList, "", "  ")

	fp := path.Join(r.Path(), IndexGZ)

	// выгрузка данных из БД в json файл
	if err = writeGzip(jsonData, fp); err != nil {
		return err
	}

	// подсчет hash суммы индекс-файла
	hash, err := HashSumFile(fp)
	if err != nil {
		return err
	}

	// запись хэш-суммы индекс-файла
	if err = writeGzipHash(fp, hash); err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

func writeGzipHash(fp, hash string) error {
	indexFileHash, err := os.Create(fp + ".sha1")
	if err != nil {
		return &internalError{
			Text:   fmt.Sprintf("ошибка создания файла %s", fp),
			Caller: "Populate",
			Err:    err,
		}
	}
	defer func() {
		_ = indexFileHash.Close()
	}()

	if _, err = indexFileHash.Write([]byte(hash)); err != nil {
		return err
	}
	return nil
}

func writeGzip(jsonData []byte, fp string) error {
	indexFile, err := os.Create(fp)
	if err != nil {
		return &internalError{
			Text:   fmt.Sprintf("ошибка создания файла %s", fp),
			Caller: "Populate::writeGzip",
			Err:    err,
		}
	}
	zw := gzip.NewWriter(indexFile)
	defer func() {
		_ = zw.Close()
		_ = indexFile.Close()
	}()

	_, err = zw.Write(jsonData)
	if err != nil {
		return &internalError{
			Text:   fmt.Sprintf("ошибка сохранения файла %s", fp),
			Caller: "Populate::writeGzip",
			Err:    err,
		}
	}
	return nil
}
