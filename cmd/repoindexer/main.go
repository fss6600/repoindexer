// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов
package main

import (
	"flag"
	"fmt"
	"log"
)

const version string = "0.0.1a"
var repoPath string
var flagVersion, flagFullMode, flagPopIndex bool

func init() {
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "repopath", "", "полный путь к репозиторию")
	flag.BoolVar(&flagFullMode, "f", false, "режим полной индексации")
	flag.BoolVar(&flagVersion, "v", false, "версия программы")
	flag.BoolVar(&flagPopIndex, "p", false, "выгрузить данные в индекс-файл после индексации")
	flag.Parse()
}

func main() {
	// вывод версии
	if flagVersion {
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
	// инициализация репозитория
	case "init":
		if err := initDB(repoPath); err != nil {
			log.Fatalf("ошибка при инициализации репозитория %v: %v", repoPath, err)
		}
		return
	// on|off режим регламента
	case "reglament":
		var mode string
		cmdRegl := flag.NewFlagSet("reglament", flag.ErrorHandling(1))
		if err := cmdRegl.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}
		if len(cmdRegl.Args()) != 0 {
			mode = cmdRegl.Arg(0)
		}
		// установка режима регламента
		setReglamentMode(repoPath, mode)
		return
	}

	// инициализация и подключение к БД
	repoPtr := NewRepoObj(repoPath)
	if err := repoPtr.OpenDB(); err != nil {
		log.Fatalln(err)
	}
	defer repoPtr.CloseDB()

	if flagFullMode {
		repoPtr.SetFullMode()
	}

	switch cmd {
	//индексация файлов репозитория с записью в БД
	case "index":
		cmdIndex := flag.NewFlagSet("index", flag.ErrorHandling(1))
		if err := cmdIndex.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}
		// индексация репозитория
		if err := index(repoPtr, cmdIndex.Args()); err != nil {
			log.Fatalln("ошибка индексирования репозитория:", err)
		}
		// flag p: выгрузка в индекс-файл
		if flagPopIndex {
			goto DOPOPULATE
		}
		break
	DOPOPULATE:
		fallthrough
	// выгрузка данных индексации из БД в index.json[gz]
	case "populate":
		if err := populate(repoPtr); err != nil {
			log.Fatalln("ошибка выгрузки индекса:", err)
		}
	// активация/деактивация пакетов в репозитории
	case "enable", "disable":
		disabled := false
		cmdSetStatus := flag.NewFlagSet("setstatus", flag.ErrorHandling(1))
		if err := cmdSetStatus.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}
		// наименования пакетов
		packetsList := cmdSetStatus.Args()
		if len(packetsList) == 0 {
			fmt.Println("укажите по крайней мере один пакет")
			return
		}
		if cmd == "disable" {
			disabled = true
		}
		if err := setPacketStatus(repoPtr, disabled, packetsList); err != nil {
			log.Fatalf("ошибка установления статуса пакетов: %v", err)
		}

	case "alias": // установка/снятие псевдонимов пакетов

	case "clean": // профилактика БД

	case "cleardb": // очистка БД от данных

	case "status": // вывод информации о репозитории
		if err := repoStatus(repoPtr); err != nil {
			log.Fatalln(err)
		}

	default:
		fmt.Println("команда не опознана")
	}
}
