package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	fnReglament string = "__REGLAMENT__"
	fnIndexName string = "index.gz"
)

func repoInit(repoPath string) error {
	fmt.Printf("инициализация репозитория: %v\n", repoPath)
	return nil
}

// Reglament активирует/деактивирует режим регламента репозитория
func Reglament(repoPath string, mode string) (string, error) {
	var (
		msg            *string
		fileExists     bool   = true
		reglOnMessage  string = "режим регламента активирован [on]"
		reglOffMessage string = "режим регламента деактивирован [off]"
	)

	fileRegl := filepath.Join(repoPath, fnReglament)
	// проверка на наличие файла-флага
	if _, err := os.Stat(fileRegl); os.IsNotExist(err) {
		fileExists = false
	}

	switch mode {
	case "":
		if fileExists {
			msg = &reglOnMessage
		} else {
			msg = &reglOffMessage
		}
	case "on":
		if fileExists {
			owner, _ := ioutil.ReadFile(fileRegl)
			v := fmt.Sprintf("%s (%s)", reglOnMessage, string(owner))
			msg = &v
			break
		}
		if err := ioutil.WriteFile(fileRegl, taskOwnerInfo(), 0644); err != nil {
			return *msg, err
		}
		msg = &reglOnMessage
	case "off":
		if fileExists {
			if err := os.Remove(fileRegl); err != nil {
				return *msg, err
			}
		}
		msg = &reglOffMessage
	default:
		return *msg, fmt.Errorf("неверный режим регламента: %s", mode)
	}
	return *msg, nil
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
