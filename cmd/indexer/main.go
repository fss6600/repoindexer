// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/proc"
)

const version string = "0.0.1a"

var repoPath string
var flagVersion, flagFullMode, flagPopIndex, flagDebug bool

func init() {
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "repopath", "", "полный путь к репозиторию")
	flag.BoolVar(&flagFullMode, "f", false, "режим полной индексации")
	flag.BoolVar(&flagVersion, "v", false, "версия программы")
	flag.BoolVar(&flagPopIndex, "p", false, "выгрузить данные в индекс-файл после индексации")
	flag.BoolVar(&flagDebug, "d", false, "режим отладки")
	flag.Parse()
}

func checkPanic() {
	if !flagDebug {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}

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
	defer checkPanic()

	// обработка команд, не требующих подключения к БД
	switch cmd {
	// инициализация репозитория
	case "init":
		if err := obj.InitDB(repoPath); err != nil {
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
		proc.SetReglamentMode(repoPath, mode)
		return
	}

	// инициализация и подключение к БД
	repoPtr := obj.NewRepoObj(repoPath)
	if err := repoPtr.OpenDB(); err != nil {
		log.Fatalln(err)
	}
	defer repoPtr.Close()

	//if flagFullMode {
	//	repoPtr.SetFullMode()
	//}

	switch cmd {
	//индексация файлов репозитория с записью в БД
	case "index":
		cmdIndex := flag.NewFlagSet("index", flag.ErrorHandling(1))
		if err := cmdIndex.Parse(flag.Args()[1:]); err != nil {
			log.Fatalln(err)
		}
		// индексация репозитория
		if err := proc.Index(repoPtr, cmdIndex.Args()); err != nil {
			log.Fatalln("ошибка индексирования репозитория:", err)
		}
		// flag p: выгрузка в индекс-файл
		if flagPopIndex {
			goto DOPOPULATE
		}
		break
	DOPOPULATE:
		fallthrough
	// выгрузка данных индексации из БД в Index.json[gz]
	case "populate":
		fmt.Println("выгрузка данных в индекс файл")
		if err := proc.Populate(repoPtr); err != nil {
			log.Fatalln("ошибка выгрузки индекса:", err)
		}
	// активация пакетов в репозитории
	case "enable":
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

		status := proc.PackStateEnable
		if err := proc.SetPackStatus(repoPtr, status, packetsList); err != nil {
			log.Fatalf("ошибка установления статуса пакетов: %v", err)
		}
	// деактивация пакетов в репозитории
	case "disable":
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

		status := proc.PackStateDisable
		if err := proc.SetPackStatus(repoPtr, status, packetsList); err != nil {
			log.Fatalf("ошибка установления статуса пакетов: %v", err)
		}

	case "alias": // установка/снятие псевдонимов пакетов
		var cmd string
		var aliases []string
		cmdAlias := flag.NewFlagSet("alias", flag.ErrorHandling(1))
		if err := cmdAlias.Parse(flag.Args()[1:]); err != nil {
			panic(fmt.Errorf(":aliases: %v", err))
		}
		if len(cmdAlias.Args()) == 0 {
			cmd = ""
			aliases = nil
		} else {
			cmd = cmdAlias.Args()[0]
			aliases = cmdAlias.Args()[1:]
		}
		proc.Alias(repoPtr, cmd, aliases)

	case "clean": // профилактика БД

	case "cleardb": // очистка БД от данных
		var cmd string
		cmdAlias := flag.NewFlagSet("cleardb", flag.ErrorHandling(1))
		if err := cmdAlias.Parse(flag.Args()[1:]); err != nil {
			panic(fmt.Errorf(":cleardb: %v", err))
		}
		if len(cmdAlias.Args()) == 0 {
			cmd = ""
		} else {
			cmd = cmdAlias.Args()[0]
		}
		proc.ClearDB(repoPtr, cmd)

	case "status": // вывод информации о репозитории
		proc.RepoStatus(repoPtr)

	default:
		fmt.Println("команда не опознана")
	}
}
