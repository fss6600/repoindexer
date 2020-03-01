// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов
package main

import (
	"flag"
	"fmt"
	"log"
)

const version string = "0.0.1a"

func main() {
	var repoPath string
	var getVersion, fullMode, popIndex bool

	flag.StringVar(&repoPath, "repopath", "", "полный путь к репозиторию")
	flag.BoolVar(&fullMode, "f", false, "режим полной индексации")
	flag.BoolVar(&getVersion, "v", false, "версия программы")
	flag.BoolVar(&popIndex, "p", false, "выгрузить данные в индекс-файл")
	flag.Parse()

	// вывод версии
	if getVersion {
		fmt.Printf("Версия: %v\n", version)
		return
	}

	// проверка на наличие пути к репозиторию
	if repoPath == "" {
		fmt.Println("не указан путь к репозиторию")
		// flag.PrintDefaults()
		return
	}

	// проверка на наличие команды и последующая обработка
	if len(flag.Args()) == 0 {
		fmt.Println("не указана команда")
		// flag.PrintDefaults()
		return
	}
	cmd := flag.Args()[0]

	// обработка команд, не требующих подключения к БД
	switch cmd {
	case "init": // инициализация репозитория
		if err := repoInit(repoPath); err != nil {
			log.Fatalf("ошибка при инициализации репозитория %v: %v", repoPath, err)
		}
		return

	case "reglament": // on|off режим регламента
		var mode string

		cmdRegl := flag.NewFlagSet("reglament", flag.ErrorHandling(1))
		if err := cmdRegl.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}

		if len(cmdRegl.Args()) != 0 {
			mode = cmdRegl.Arg(0)
		}

		if msg, err := Reglament(repoPath, mode); err != nil {
			log.Fatalf("ERROR: %v\n", err)
		} else {
			fmt.Println(msg)
		}
		return
	}

	// подключение к БД
	pRepoObj, err := NewRepoObj(repoPath)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	if fullMode {
		pRepoObj.SetFullMode()
	}

	defer pRepoObj.Close()

	switch cmd {
	case "index": // индексация файлов репозитория с записью в БД
		cmdIndex := flag.NewFlagSet("index", flag.ErrorHandling(1))
		if err := cmdIndex.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}

		if err := index(pRepoObj, cmdIndex.Args()); err != nil {
			log.Fatalf("ошибка индексирования репозитория: %v\n", err)
		}
		if popIndex {
			goto DO_POPULATE // выгрузка данных в индекс-файл
		}
		break
	DO_POPULATE:
		fallthrough
	case "populate": // выгрузка данных индексации из БД в index.json[gz]
		if err = populate(pRepoObj); err != nil {
			log.Fatalf("ошибка выгрузки данных: %v\n", err)
		}

	case "enable", "disable": // активация/деактивация пакетов в репозитории
		var setDisable bool
		cmdSetStatus := flag.NewFlagSet("setstatus", flag.ErrorHandling(1))
		if err := cmdSetStatus.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}

		packs := cmdSetStatus.Args()

		if len(packs) == 0 {
			fmt.Println("укажите по крайней мере один пакет")
			return
		}

		if cmd == "disable" {
			setDisable = true
		}

		if err := setPacketStatus(pRepoObj, setDisable, packs); err != nil {
			log.Fatalf("ошибка установления статуса пакетов: %v", err)
		}

	case "alias": // установка/снятие псевдонимов пакетов

	case "clean": // профилактика БД

	case "cleardb": // очистка БД от данных

	case "status": // вывод информации о репозитории
		if err := repoStatus(pRepoObj); err != nil {
			log.Fatalln(err)
		}

	default:
		fmt.Println("команда не опознана")
	}
}
