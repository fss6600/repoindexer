// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов
package main

import (
	"flag"
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/proc"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const version string = "0.0.2a"

var repoPath string
var flagPopIndex, flagDebug bool

func init() {
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "r", "", "repopath: полный путь к репозиторию")
	flag.BoolVar(&flagPopIndex, "p", false, "populate: выгрузить данные в индекс-файл после индексации")
	flag.BoolVar(&flagDebug, "d", false, "debug: режим отладки")
	flag.Parse()
}

// обработка вызова panic в любой части программы
func checkPanic() {
	if !flagDebug {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}

}

func main() {
	const tmplErrMsg = "error::main:"
	// отложенная обработка сообщений об ошибках
	defer checkPanic()

	// проверка на наличие пути к репозиторию
	if repoPath == "" {
		panic("не указан путь к репозиторию")
		// flag.PrintDefaults()
	}
	// проверка на наличие команды и последующая обработка
	if len(flag.Args()) == 0 {
		panic("не указана команда")
		// flag.PrintDefaults()
	}
	cmd := flag.Args()[0]

	// обработка команд, не требующих подключения к БД
	switch cmd {
	// инициализация репозитория
	case "init":
		err := obj.InitDB(repoPath)
		utils.CheckError(fmt.Sprintf("ошибка при инициализации репозитория %v", repoPath), &err)
		return // выходим, чтобы не инициализировать подключение к БД
	// on|off режим регламента
	case "reglament":
		var mode string
		cmdRegl := flag.NewFlagSet("reglament", flag.ErrorHandling(1))
		err := cmdRegl.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		if len(cmdRegl.Args()) != 0 {
			mode = cmdRegl.Arg(0)
		}
		// установка режима регламента
		proc.SetReglamentMode(repoPath, mode)
		return // выходим, чтобы не инициализировать подключение к БД
	}

	// инициализация и подключение к БД
	repoPtr, err := obj.NewRepoObj(repoPath)
	utils.CheckError(tmplErrMsg, &err)
	err = repoPtr.OpenDB()
	utils.CheckError(tmplErrMsg, &err)
	defer func() {
		err := repoPtr.Close()
		utils.CheckError(tmplErrMsg, &err)
	}()

	switch cmd {
	//индексация файлов репозитория с записью в БД
	case "index":
		cmdIndex := flag.NewFlagSet("index", flag.ErrorHandling(1))
		err := cmdIndex.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		proc.Index(repoPtr, cmdIndex.Args())
		// flag p: при указании - выгрузка в индекс-файл
		if flagPopIndex {
			goto DOPOPULATE
		}
		break
	DOPOPULATE:
		fallthrough
	// выгрузка данных индексации из БД в Index.json[gz]
	case "populate":
		fmt.Println("выгрузка данных в индекс файл")
		proc.Populate(repoPtr)
	// активация/блокировка пакетов в репозитории
	case "enable", "disable":
		cmdSetStatus := flag.NewFlagSet("setstatus", flag.ErrorHandling(1))
		err := cmdSetStatus.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		// список пакетов из командной строки
		packetsList := cmdSetStatus.Args()
		if len(packetsList) == 0 {
			panic("укажите по крайней мере один пакет")
		}
		var status proc.PackStatus
		if cmd == "enable" {
			status = proc.PackStateEnable
		} else {
			status = proc.PackStateDisable
		}
		proc.SetPackStatus(repoPtr, status, packetsList)
	// присвоение/удаление/отображение псевдонимов
	case "alias":
		var cmd string
		var aliases []string
		cmdAlias := flag.NewFlagSet("alias", flag.ErrorHandling(1))
		err := cmdAlias.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		if len(cmdAlias.Args()) == 0 {
			cmd = ""
			aliases = nil
		} else {
			cmd = cmdAlias.Args()[0]
			aliases = cmdAlias.Args()[1:]
		}
		proc.Alias(repoPtr, cmd, aliases)
	// вывод перечня и статус пакетов в репозитории
	case "list":
		var cmd string
		cmdAlias := flag.NewFlagSet("list", flag.ErrorHandling(1))
		err := cmdAlias.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		if len(cmdAlias.Args()) == 0 {
			cmd = "all"
		} else {
			cmd = cmdAlias.Args()[0]
		}
		proc.List(repoPtr, cmd)
	// вывод версии программы, БД
	case "version":
		vMaj, vMin, err := repoPtr.VersionDB()
		utils.CheckError(tmplErrMsg, &err)
		fmt.Printf("Версия программы\t: %v\n", version)
		fmt.Printf("Версия БД программы\t: %d.%d\n", obj.DBVersionMajor, obj.DBVersionMinor)
		fmt.Printf("Версия БД репозитория\t: %d.%d\n", vMaj, vMin)
	// упаковка и переиндексация данных в БД
	case "clean":
		fmt.Print("упаковка БД: ")
		err = repoPtr.Clean()
		utils.CheckError(tmplErrMsg, &err)
		fmt.Println("OK")
	// очистка БД от данных
	case "cleardb":
		var cmd string
		cmdAlias := flag.NewFlagSet("cleardb", flag.ErrorHandling(1))
		err := cmdAlias.Parse(flag.Args()[1:])
		utils.CheckError(tmplErrMsg, &err)
		if len(cmdAlias.Args()) == 0 {
			cmd = ""
		} else {
			cmd = cmdAlias.Args()[0]
		}
		proc.ClearDB(repoPtr, cmd)
	// вывод информации о репозитории
	case "status":
		proc.RepoStatus(repoPtr)
	default:
		panic("команда не опознана")
	}
}
