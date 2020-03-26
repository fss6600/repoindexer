package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/proc"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

var repoPath string
var flagPopIndex, flagDebug bool

const tmplErrMsg = "main:"

func init() {
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "r", "", "repopath: полный путь к репозиторию")
	flag.BoolVar(&flagPopIndex, "p", false, "populate: выгрузить данные в индекс-файл после индексации")
	flag.BoolVar(&flagDebug, "d", false, "debug: режим отладки")
	flag.Parse()
}
func Run() {
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
		utils.CheckError(fmt.Sprintf("ошибка при инициализации репозитория '%v':", repoPath), &err)
		return // выходим, чтобы не инициализировать подключение к БД
	// on|off режим регламента
	case "regl", "reglament":
		var mode string
		cmdRegl := newFlagSet("reglament")
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
	utils.CheckError("", &err)
	defer func() {
		err := repoPtr.Close()
		utils.CheckError(tmplErrMsg, &err)
	}()

	switch cmd {
	//индексация файлов репозитория с записью в БД
	case "index":
		cmdIndex := newFlagSet("index")
		packs := cmdIndex.Args() // из командной строки
		if len(packs) == 0 {
			packs = readFromStdin() // из stdin
		}
		if len(packs) == 0 {
			packs = repoPtr.ActivePacks() // активные
		}
		proc.Index(repoPtr, packs)
		// flag p: при указании - выгрузка в индекс-файл
		if flagPopIndex {
			goto DOPOPULATE
		}
		break
	DOPOPULATE:
		fallthrough
	// выгрузка данных индексации из БД в Index.json[gz]
	case "pop", "populate":
		proc.Populate(repoPtr)
	// активация/блокировка пакетов в репозитории
	case "enable", "disable":
		cmdSetStatus := newFlagSet("setstatus")
		// список пакетов из командной строки
		packetsList := cmdSetStatus.Args()
		if len(packetsList) == 0 {
			// список пакетов из stdin
			packetsList = readFromStdin()
			if len(packetsList) == 0 {
				panic("укажите по крайней мере один пакет")
			}
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
		cmdAlias := newFlagSet("alias")
		if len(cmdAlias.Args()) == 0 {
			cmd = "show"
			aliases = nil
		} else {
			cmd = cmdAlias.Args()[0]
			aliases = cmdAlias.Args()[1:]
			if len(aliases) == 0 {
				// from stdin
				aliases = readFromStdin()
			} else if len(aliases) == 0 {
				panic("укажите по крайней мере 1 пару ПАКЕТ=ПСЕВДОНИМ")
			}
		}
		proc.Alias(repoPtr, cmd, aliases)
	// вывод перечня и статус пакетов в репозитории
	case "list":
		var cmd string
		cmdAlias := newFlagSet("list")
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
		fmt.Print("Упаковка БД: ")
		err = repoPtr.Clean()
		utils.CheckError(tmplErrMsg, &err)
		fmt.Println("OK")
	// очистка БД от данных
	case "cleardb":
		var cmd string
		cmdAlias := newFlagSet("cleardb")
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

func readFromStdin() []string {
	ch := make(chan string)
	go func(chan string) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
	}(ch)

	lst := make([]string, 0)

	for {
		select {
		case input, more := <-ch:
			if more {
				lst = append(lst, input)
			} else {
				return lst
			}
		case <-time.After(time.Millisecond * 50):
			return lst
		}
	}
}

func newFlagSet(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ErrorHandling(1))
	err := f.Parse(flag.Args()[1:])
	utils.CheckError(tmplErrMsg, &err)
	return f
}
