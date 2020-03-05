package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const (
	fnReglament string = "__REGLAMENT__"
	//fnIndexName string = "index.gz"
)

// SetReglamentMode активирует/деактивирует режим регламента репозитория
func SetReglamentMode(repoPath, mode string) {
	const (
		reglOnMessage  string = "режим регламента активирован [on]"
		reglOffMessage string = "режим регламента деактивирован [off]"
	)
	fRegl := filepath.Join(repoPath, fnReglament)
	// проверка на наличие файла-флага, определение режима реглавмета
	modeOn := fileExists(fRegl)

	switch mode {
	// вывод режима регламента
	case "":
		if modeOn {
			fmt.Println(reglOnMessage)
		} else {
			fmt.Println(reglOffMessage)
		}
	// активация режима регламента
	case "on":
		// регламент уже активирован - вывод сообщения и информации кто активировал
		if modeOn {
			owner, _ := ioutil.ReadFile(fRegl)
			fmt.Println(reglOnMessage, string(owner))
			// авктивация реглавмента с записью информации кто активировал
		} else {
			if err := ioutil.WriteFile(fRegl, taskOwnerInfo(), 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Println(reglOnMessage)
		}
	// деактивация режима реглавмента
	case "off":
		// регламент активирован - удаляем файл
		if modeOn {
			if err := os.Remove(fRegl); err != nil {
				log.Fatalln(err)
			}
		}
		fmt.Println(reglOffMessage)
	default:
		fmt.Printf("неверный режим регламента: %s", mode)
	}
}

// Index обработка и индексация пакетов в репозитории
func Index(r *RepoObject, packs []string) error {
	if len(packs) == 0 {
		// get active packs list
		packs = r.ActivePacks()
	}

	for _, pack := range packs {
		if err := processPacketIndex(r, pack); err != nil {
			return err
		}
	}
	return nil
}

func Populate(repo *RepoObject) error {
	{
		fmt.Println("выгрузка данных из БД в Index файл")
	}
	return nil
}

func RepoStatus(repo *RepoObject) error {
	fmt.Println("вывод информации о репозитории")
	return nil
}

// SetPacketStatus активирует или блокирует пакет для индексации
func SetPacketStatus(repo *RepoObject, setDisable bool, pack []string) error {
	if setDisable {
		fmt.Printf("деактивация пакетов: %s\n", pack)
	} else {
		fmt.Printf("активация пакетов: %s\n", pack)
	}

	return nil
}

// dbl - структура данных о файле в пакете в БД
type dbl struct {
	id    int
	path  string
	size  int
	mdate int // todo: change to datetime
}

// processPacketIndex обрабатывает (индексирует) файлы в указанном пакете
func processPacketIndex(r *RepoObject, pack string) error {
	var (
		fsi, dbi int      // счетчики для обхода списков
		fsList   []string // список путей файлов в пакете в репозитории
		dbList   []dbl    // список данных файлов в пакете из БД
		fsp      string   // путь файла в репозитории
		dbp      dbl      // данные о файле в БД
	)

	// получение списка файлов в пакете (path)
	// получение списка с данными о файлах пакета из БД (id, path, size, mdate)

	// todo пересмотреть алгоритм сравнения файлов

	// цикл обработки данных
	for {
		fsp = fsList[fsi] // todo add error handle
		dbp = dbList[dbi]

		break
	}

	return nil
}

// возвращает данные IP,.. инициатора работ в репозитории
func taskOwnerInfo() []byte {
	return []byte("127.0.0.1") //todo: добавить информацию о подключении
}
