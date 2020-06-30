package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/proc"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

var repoPath string
var flagFullIndex, flagDebug, flagVersion bool

const tmplErrMsg = "main:"

// STDINWAIT период времени для таймера ожидания ввода с stdin
const STDINWAIT = time.Millisecond * 50

type conf struct {
	Repo string `json:"repo"`
}

func init() {
	var rp string
	if cnf, err := readConfFromJSON(); err == nil {
		rp = cnf.Repo
	}
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "r", rp, "*полный путь к репозиторию")
	flag.BoolVar(&flagDebug, "d", false, "режим отладки")
	flag.BoolVar(&flagFullIndex, "f", false, "режим принудительной полной индексации")
	flag.BoolVar(&flagVersion, "v", false, "версия программы")
	flag.Usage = usage

	flag.Parse()
}

// Run обрабатывает команды командной строки
func Run() {
	if flagVersion {
		fmt.Println("Версия:", version)
		return
	}

	// проверка на наличие пути к репозиторию
	if repoPath == "" {
		panic("не указан путь к репозиторию")
	}

	// проверка на наличие команды и последующая обработка
	if len(flag.Args()) == 0 {
		panic("не указана команда")
	}

	fmt.Println("установлен путь к репозиторию:", repoPath)
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
	pRepo, err := obj.NewRepoObj(repoPath)
	utils.CheckError(tmplErrMsg, &err)
	err = pRepo.OpenDB()
	utils.CheckError("", &err)
	defer func() {
		err := pRepo.Close()
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
			packs = pRepo.ActivePacks() // активные
		}
		proc.Index(pRepo, flagFullIndex, packs)

	// выгрузка данных индексации из БД в Index.gz
	case "pop", "populate":
		proc.Populate(pRepo)

	// обработка исполняемых файлов пакетов
	case "exec":
		var cmd string
		var packs []string
		cmdExecFile := newFlagSet("execfile")

		if len(cmdExecFile.Args()) == 0 {
			panic("укажите одну из команд: check | set | del | show")
		}
		cmd = cmdExecFile.Args()[0]
		packs = cmdExecFile.Args()[1:]
		proc.ExecFile(pRepo, cmd, packs)

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
		if cmd == "enable" {
			proc.SetPackStatus(pRepo, obj.PackStatusActive, packetsList)
		} else {
			proc.SetPackStatus(pRepo, obj.PackStatusBlocked, packetsList)
		}

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
				if len(aliases) == 0 {
					panic("укажите по крайней мере 1 пару ПАКЕТ=ПСЕВДОНИМ")
				}
			}
		}
		proc.Alias(pRepo, cmd, aliases)

	// вывод перечня и статус пакетов в репозитории
	case "list":
		var cmd string
		cmdList := newFlagSet("list")
		if len(cmdList.Args()) == 0 {
			cmd = "all"
		} else {
			cmd = cmdList.Args()[0]
		}
		proc.List(pRepo, cmd)

	// упаковка и переиндексация данных в БД
	case "clean":
		fmt.Print("Упаковка БД: ")
		err = pRepo.Clean()
		utils.CheckError(tmplErrMsg, &err)
		fmt.Println("OK")

	// очистка БД от данных
	case "cleardb":
		var cmd string
		cmdClearDB := newFlagSet("cleardb")
		if len(cmdClearDB.Args()) == 0 {
			cmd = ""
		} else {
			cmd = cmdClearDB.Args()[0]
		}
		proc.ClearDB(pRepo, cmd)

	// вывод информации о репозитории
	case "status":
		proc.RepoStatus(pRepo)

	// миграция БД
	case "migrate":
		proc.MigrateDB(pRepo)

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
		case <-time.After(STDINWAIT):
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

// проверка файла-конфигурации ,чтение настроек
func readConfFromJSON() (cnf conf, err error) {
	curFilePath, _ := os.Executable()
	curDir := filepath.Dir(curFilePath)
	files, _ := filepath.Glob(filepath.Join(curDir, "*.conf"))

	if len(files) > 0 {
		confFile := filepath.Clean(files[0])
		buf, err := ioutil.ReadFile(confFile)
		if err != nil {
			return cnf, err
		}
		err = json.Unmarshal(buf, &cnf)

		switch err.(type) {
		case *json.SyntaxError:
			fmt.Println("неверный синтаксис файла настроек", err)
		}
	}
	return cnf, err
}
