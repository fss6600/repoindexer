package handler

import (
	"fmt"
)

// TODO: алгоритм миграции с сохранением данных БД

// MigrateDB обрабатывает команду `migrate`
// сохраняет данные псевдонимов и блокировок пакетов при миграции
// и импортирует обратно после удаления данных из БД
// Миграция требуется при изменении структуры БД
func MigrateDB(r *Repo) error {
	checkRegl(r.path)
	if !UserAccept("\nДанная операция заменит файлы БД и индекса." +
		"\nУбедитесь, что у Вас есть резервная копия") {
		return nil
	}
	fmt.Println()
	vMaj, vMin, err := r.VersionDB()
	if err != nil {
		return err
	}
	if DBVersionMajor > vMaj {
		return &InternalError{
			Text:   "\n\tтребуется переиндексация репозитория",
			Caller: "Migrate",
		}
	} else if DBVersionMajor < vMaj || DBVersionMinor < vMin {
		return &InternalError{
			Text:   "возможно вы используете старую версию программы",
			Caller: "Migrate",
		}
	} else if DBVersionMajor == vMaj && DBVersionMinor == vMin {
		fmt.Println("миграция не требуется")
		return nil
	}
	tmpl := "%-30s: "
	//подготовка списка заблокированных пакетов
	fmt.Printf(tmpl, "сохранение alias, blocked")
	blocked := r.DisabledPacks()
	//подготовка списка псевдонимов
	aliases := r.Aliases()
	fmt.Println("OK")

	// close DB
	fmt.Printf(tmpl, "Закрытие БД")
	if err = r.Close(); err != nil {
		return err
	}
	fmt.Println("OK")

	// очистка старых файлов БД и индекса
	fmt.Printf(tmpl, "удаление файлов БД и индекса")
	if err = CleanForMigrate(r); err != nil {
		return err
	}
	fmt.Println("OK")

	// инициализация репозитория
	fmt.Printf(tmpl, "инициализация репозитория")
	if err = InitDB(r.Path()); err != nil {
		return err
	}

	// connect to DB
	if err = r.OpenDB(); err != nil {
		return err
	}

	// ввод псевдонимов в БД
	fmt.Println("восстановление alias, blocked")
	for _, alias := range aliases {
		if err = r.SetAlias(alias); err != nil {
			return err
		}
	}

	// ввод заблокированных в БД
	for _, pack := range blocked {
		if err = r.DisablePack(pack); err != nil {
			return err
		}
	}
	fmt.Println("Миграция завершена")
	fmt.Println("\n\tТребуется индексация репозитория")
	return nil
}
