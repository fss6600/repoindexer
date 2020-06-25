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
var flagFullIndex, flagPopIndex, flagDebug, flagVersion bool

const tmplErrMsg = "main:"

func init() {
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "r", "", "repopath: полный путь к репозиторию")
	flag.BoolVar(&flagDebug, "d", false, "debug: режим отладки")
	flag.BoolVar(&flagFullIndex, "f", false, "full: режим принудительной полной индексации")
	flag.BoolVar(&flagVersion, "v", false, "version: версия программы")
	flag.Parse()
}

func Run() {
	if flagVersion {
		fmt.Printf("Версия программы\t: %v\n", version)
		return
	}

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
			packs = readDataFromStdin() // из stdin через pipe
		}
		if len(packs) == 0 {
			packs = repoPtr.ActivePacks() // активные
		}
		proc.Index(repoPtr, flagFullIndex, packs)
	// выгрузка данных индексации из БД в Index.json[gz]
	case "pop", "populate":
		proc.Populate(repoPtr)
	//
	case "exec":
		var cmd string
		var packs []string
		cmdExecFile := newFlagSet("execfile")

		if len(cmdExecFile.Args()) == 0 {
			panic("укажите одну из команд: check | set | del | show")
		}
		cmd = cmdExecFile.Args()[0]
		packs = cmdExecFile.Args()[1:]
		proc.ExecFile(repoPtr, cmd, packs)
	// активация/блокировка пакетов в репозитории
	case "enable", "disable":
		cmdSetStatus := newFlagSet("setstatus")
		// список пакетов из командной строки
		packetsList := cmdSetStatus.Args()
		if len(packetsList) == 0 {
			// список пакетов из stdin
			packetsList = readDataFromStdin()
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
				aliases = readDataFromStdin()
			} else {
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
	// упаковка и переиндексация данных в БД
	case "clean":
		fmt.Print("Упаковка БД: ")
		err = repoPtr.Clean()
		utils.CheckError(tmplErrMsg, &err)
		fmt.Println("OK")
	// очистка БД от данных
	case "cleardb":
		// todo добавить подтверждение
		if !utils.UserAccept("Данная операция удаляет данные из БД") {
			return
		}
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
	case "migrate":
		proc.MigrateDB(repoPtr)
	default:
		panic("команда не опознана")
	}
}

func readDataFromStdin() []string {
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
