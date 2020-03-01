package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	fnReglament string = "__REGLAMENT__"
	//fnIndexName string = "index.gz"
)

func repoInit(repoPath *string) error {
	fmt.Printf("инициализация репозитория: %v\n", *repoPath)
	return nil
}

// reglamentMode активирует/деактивирует режим регламента репозитория
func reglamentMode(repoPath *string, mode *string) (string, error) {
	const (
		reglOnMessage  string = "режим регламента активирован [on]"
		reglOffMessage string = "режим регламента деактивирован [off]"
	)

	var fileExists bool = true

	fileRegl := filepath.Join(*repoPath, fnReglament)
	// проверка на наличие файла-флага
	if _, err := os.Stat(fileRegl); os.IsNotExist(err) {
		fileExists = false
	}

	switch *mode {
	case "":
		if fileExists {
			return reglOnMessage, nil
		} else {
			return reglOffMessage, nil
		}
	case "on":
		if fileExists {
			owner, _ := ioutil.ReadFile(fileRegl)
			return fmt.Sprintf("%s (%s)", reglOnMessage, string(owner)), nil
		}

		if err := ioutil.WriteFile(fileRegl, taskOwnerInfo(), 0644); err != nil {
			return "", err
		}
		return reglOnMessage, nil
	case "off":
		if fileExists {
			if err := os.Remove(fileRegl); err != nil {
				return "", err
			}
		}
		return reglOffMessage, nil
	default:
		return "", fmt.Errorf("неверный режим регламента: %s", mode)
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
	return []byte("127.0.0.1\n") //todo: добавить информацию о подключении
}
