package main

import (
	"flag"
	"fmt"
	"log"
)

const (
	version string = "0.0.1a"
	fDBName string = "index.db"
	fIndexName string = "index.gz"
	fReglament string = "__REGLAMENT__"
)

// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов

func main() {
	// 
	repoPath := flag.String("repopath", "", "полный путь к репозиторию")
	// fullMode := flag.Bool("full", false, "режим полной индексации")
	getVersion := flag.Bool("version", false, "версия программы")

	flag.Parse()

	// вывод версии
	if *getVersion {
		fmt.Printf("Версия: %v\n", version)
		return
	}

	// проверка на наличие команды
	if len(flag.Args()) == 0 {
		fmt.Println("не указана команда")
		// flag.PrintDefaults()
		return
	}

	// проверка на наличие пути к репозиторию
	if *repoPath == "" {
		fmt.Println("не указан путь к репозиторию")
		// flag.PrintDefaults()
		return
	}

	// fmt.Println(flag.Args())

	switch flag.Args()[0] {
	case "init":
		fmt.Println("инициализация репозитория")

	case "populate": {
		fmt.Println("выгрузка данных из БД в index файл")
	}

	case "index":
		cmdIndex := flag.NewFlagSet("index", flag.ErrorHandling(1))
		cmdIndex.Parse(flag.Args()[1:])
		if err := index(repoPath, cmdIndex.Args()); !=nil {
			log.Fatal("ошибка индексирования репозитория")
		}

	default:
		fmt.Println("не указана команда")
		return
	}
}