package main

import (
	"flag"
	"fmt"
	"os"
)

var Usage = func() {
	fmt.Printf("Использование программы: %s [флаг] команда [параметр команды, ...]\n", os.Args[0])
	PrintUsage()
}

func PrintUsage() {
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
