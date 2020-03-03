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

// setReglamentMode активирует/деактивирует режим регламента репозитория
func setReglamentMode(repoPath, mode string) {
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

func index(repo *RepoObject, packets []string) error {
	if len(packets) == 0 {
		fmt.Println("обработка всех пакетов")
	} else {
		for _, pack := range packets {
			fmt.Printf("обработка пакета: %v\n", pack)
		}
	}
	return nil
}

func populate(repo *RepoObject) error {
	{
		fmt.Println("выгрузка данных из БД в index файл")
	}
	return nil
}

func repoStatus(repo *RepoObject) error {
	fmt.Println("вывод информации о репозитории")
	return nil
}

func setPacketStatus(repo *RepoObject, setDisable bool, pack []string) error {
	if setDisable {
		fmt.Printf("деактивация пакетов: %s\n", pack)
	} else {
		fmt.Printf("активация пакетов: %s\n", pack)
	}

	return nil
}

// возвращает данные IP,.. инициатора работ в репозитории
func taskOwnerInfo() []byte {
	return []byte("127.0.0.1") //todo: добавить информацию о подключении
}
