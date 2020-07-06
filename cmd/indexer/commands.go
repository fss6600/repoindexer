package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	h "github.com/pmshoot/repoindexer/internal/handler"
)

var (
	err                                   error
	repoPath                              string
	flagFullIndex, flagDebug, flagVersion bool
)

// STDINWAIT период времени для таймера ожидания ввода с stdin
const STDINWAIT = time.Millisecond * 50

type conf struct {
	Repo string `json:"repo"`
}

func init() {
	log.SetFlags(0)
	var rp string
	if cnf, err := readConfFromJSON(); err == nil {
		rp = cnf.Repo
	} else {
		fmt.Println(err)
	}
	// обработка флагов и переменных
	flag.StringVar(&repoPath, "r", rp, "*полный путь к репозиторию")
	flag.BoolVar(&flagDebug, "d", false, "режим отладки")
	flag.BoolVar(&flagFullIndex, "f", false, "режим принудительной полной индексации")
	flag.BoolVar(&flagVersion, "v", false, "версия программы")
	flag.Usage = usage
	flag.Parse()
}

// run обрабатывает команды командной строки
func run() {
	if flagVersion {
		fmt.Println("Версия:", version)
		return
	}

	// проверка на наличие пути к репозиторию
	if repoPath == "" {
		log.Fatalln("не указан путь к репозиторию")
	}

	// проверка на наличие команды и последующая обработка
	if len(flag.Args()) == 0 {
		log.Fatalln("не указана команда")
	}

	fmt.Println("репозиторий:", repoPath)

	// обработка команд, не требующих подключения к БД
	cmd := flag.Args()[0]
	switch cmd {
	// инициализация репозитория
	case "init":
		if err = h.InitDB(repoPath); err != nil {
			fatal(err)
		}
		return // выходим, чтобы не инициализировать подключение к БД

	// on|off режим регламента
	case "regl", "reglament":
		var mode string
		cmdRegl := newFlagSet("reglament")
		if len(cmdRegl.Args()) != 0 {
			mode = cmdRegl.Arg(0)
		}
		// установка режима регламента
		if err = h.SetReglamentMode(repoPath, mode); err != nil {
			fatal(err)
		}
		return // выходим, чтобы не инициализировать подключение к БД
	}

	// инициализация и подключение к БД
	pRepo, err := h.NewRepo(repoPath)
	if err != nil {
		fatal(err)
	}
	if err = pRepo.OpenDB(); err != nil {
		fatal(err)
	}
	defer func() {
		if err = pRepo.Close(); err != nil {
			fatal(err)
		}
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
		if err = h.Index(pRepo, flagFullIndex, packs); err != nil {
			fatal(err)
		}

	// выгрузка данных индексации из БД в Index.gz
	case "pop", "populate":
		if err = h.Populate(pRepo); err != nil {
			fatal(err)
		}

	// обработка исполняемых файлов пакетов
	case "exec":
		var cmd string
		var packs []string
		cmdExecFile := newFlagSet("execfile")

		if len(cmdExecFile.Args()) == 0 {
			log.Fatal("укажите одну из команд: check | set | del | show")
		}
		cmd = cmdExecFile.Args()[0]
		packs = cmdExecFile.Args()[1:]
		if err = h.ExecFile(pRepo, cmd, packs); err != nil {
			fatal(err)
		}

	// активация/блокировка пакетов в репозитории
	case "enable", "disable":
		cmdSetStatus := newFlagSet("setstatus")
		// список пакетов из командной строки
		packetsList := cmdSetStatus.Args()
		if len(packetsList) == 0 {
			// список пакетов из stdin
			packetsList = readDataFromStdin()
			if len(packetsList) == 0 {
				log.Fatal("укажите по крайней мере один пакет")
			}
		}
		if cmd == "enable" {
			err = h.SetPackStatus(pRepo, h.PackStatusActive, packetsList)
		} else {
			err = h.SetPackStatus(pRepo, h.PackStatusBlocked, packetsList)
		}
		if err != nil {
			fatal(err)
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
					log.Fatal("укажите по крайней мере 1 пару ПАКЕТ=ПСЕВДОНИМ")
				}
			}
		}
		if err = h.Alias(pRepo, cmd, aliases); err != nil {
			fatal(err)
		}

	// вывод перечня и статус пакетов в репозитории
	case "list":
		var cmd string
		cmdList := newFlagSet("list")
		if len(cmdList.Args()) == 0 {
			cmd = "all"
		} else {
			cmd = cmdList.Args()[0]
		}
		if err = h.List(pRepo, cmd); err != nil {
			fatal(err)
		}

	// упаковка и переиндексация данных в БД
	case "clean":
		fmt.Print("Упаковка БД: ")
		if err = pRepo.Clean(); err != nil {
			fmt.Println()
			fatal(err)
		}
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
		if err = h.ClearDB(pRepo, cmd); err != nil {
			fatal(err)
		}

	// вывод информации о репозитории
	case "status":
		if err = h.RepoStatus(pRepo); err != nil {
			fatal(err)
		}

	// миграция БД
	case "migrate":
		if err = h.MigrateDB(pRepo); err != nil {
			fatal(err)
		}

	default:
		log.Fatal("команда не опознана")
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
	if err = f.Parse(flag.Args()[1:]); err != nil {
		log.Fatalf("ошибка установки flagset %v", err)
	}
	return f
}

// проверка файла-конфигурации ,чтение настроек
func readConfFromJSON() (conf, error) {
	curFilePath, _ := os.Executable()
	curDir := filepath.Dir(curFilePath)
	files, _ := filepath.Glob(filepath.Join(curDir, "*.conf"))
	var cnf conf
	if len(files) > 0 {
		confFile := filepath.Clean(files[0])
		buf, err := ioutil.ReadFile(confFile)
		if err != nil {
			return cnf, err
		}
		if err = json.Unmarshal(buf, &cnf); err != nil {
			switch err.(type) {
			case *json.SyntaxError:
				fmt.Println("неверный синтаксис файла настроек", err)
			default:
				return cnf, fmt.Errorf("ошибка чтения конфигурации - %v", err)
			}
		}
	}
	return cnf, nil
}

var usage = func() {
	fmt.Printf("Использование программы: %s [флаг] команда [параметр команды, ...]\n", os.Args[0])
	printUsage()
}

func printUsage() {
	flag.PrintDefaults()
	fmt.Println("* - обязательные флаги")
	fmt.Println("\nКоманды:")

	commands := [][]string{
		{"init", "инициализация репозитория"},
		{"regl [on|off]", "статус, активация, деактивация режима регламента"},
		{"index [packname, ...]", "индексирование репозитория или указанных пакетов"},
		{"exec [check|set|del|show [packname]]", "поиск, установка, удаление, вывод исполняемого файла для пакета[ов]"},
		{"pop", "выгрузка данных в индекс-файл"},
		{"enable packname [packname, ...] | <(stdin)", "активация заблокированного пакета[ов] "},
		{"disable packname [packname, ...] | <(stdin)", "блокировка пакета[ов]"},
		{"alias [show] | [set packname=alias,... | <(stdin)] | [del alias,... | <(stdin)]]", "вывод, установка, удаление псевдонимов для пакетов"},
		{"list", "вывод перечня и статуса пакетов в репозитории"},
		{"status", "вывод информации о состоянии репозитория"},
		{"migrate", "миграция данных БД при изменениии версии"},
		{"clean", "упаковка и переиндексация данных в БД"},
		{"cleardb index|alias|status|all", "очистка БД от данных индекса, псевдонимов, блокировок или всех данных"},
	}

	for _, comm := range commands {
		fmt.Printf("  %v\n  - %v\n", comm[0], comm[1])
	}
}

func fatal(e error) {
	switch e.(type) {
	case *h.InternalError:
		if flagDebug {
			msg := fmt.Sprintf("%s\n", e.(*h.InternalError).Text)
			msg += fmt.Sprintf("Caller: %s\n", e.(*h.InternalError).Caller)
			msg += fmt.Sprintf("Original error: %v\n", e.(*h.InternalError).Err)
			log.Fatal(msg)
		}
	}
	log.Fatalln(e)
}
