package handler

import (
	"fmt"
)

// ClearDB обрабатывает команду cleardb
// all - удаляет все данные из БД
// index - удаляет данные об индексации из БД
// alias - удаляет данные о псевдонимах из БД
// status - удаляет данные о блокировках из БД
func ClearDB(r *Repo, cmd string) error {
	var msg string
	if cmd == "all" {
		msg = "Удаляются все данные репозитория из БД"
	} else {
		msg = "Удаляются данные индекса из БД"
	}
	switch cmd {
	case "index", "all":
		if !UserAccept(msg) {
			return nil
		}
		fmt.Print("Очистка данных индексации пакетов...")
		if err = r.DBCleanPackages(); err != nil {
			return err
		}
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "alias":
		if cmd != "all" && !UserAccept("Удаляются данные псевдонимов из БД") {
			return nil
		}
		fmt.Print("Очистка данных псевдонимов...")
		if err = r.DBCleanAliases(); err != nil {
			return err
		}
		fmt.Println("OK")
		if cmd != "all" {
			break
		}
		fallthrough
	case "status":
		if cmd != "all" && !UserAccept("Удаляются данные блокировок из БД") {
			return nil
		}
		fmt.Print("Очистка данных блокировки...")
		if err = r.DBCleanStatus(); err != nil {
			return err
		}
		fmt.Println("OK")
	default:
		return &InternalError{
			Text:   "укажите одну категорию из списка: index | alias | status | all",
			Caller: "ClearDB",
		}
	}
	if cmd == "alias" {
		fmt.Println(doPopMsg)
	} else {
		fmt.Println(doIndexMsg)
	}
	return nil
}
