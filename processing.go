package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	fReglament string = "__REGLAMENT__"
	fIndexName string = "index.gz"
)

func repoInit(repoPath string) error {
	fmt.Printf("инициализация репозитория: %v\n", repoPath)
	return nil
}

func reglament(repoPath string, mode string) error {
	fileRegl := filepath.Join(repoPath, fReglament)

	switch mode {
	case "":
		if _, err := os.Stat(fileRegl); err != nil {
			fmt.Println("Режим регламента активирован [on]")
		} else {
			fmt.Println("Режим регламента отключен [off]")
		}
	case "on":
		fmt.Println("режим регламента активирован")
	case "off":
		fmt.Println("режим регламента деактивирован")
	default:
		return fmt.Errorf("неверный режим регламента: %s", mode)
	}
	return nil
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
