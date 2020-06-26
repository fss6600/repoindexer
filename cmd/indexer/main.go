// Проект "Репозиторий подсистем ЕИИС Соцстрах"
// Indexer: создание репозитория подсистем ЕИИС "Соцстрах" и индексация пакетов
package main

import (
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const version string = "0.1.2"

func main() {
	// отложенная обработка сообщений об ошибках
	defer utils.CheckPanic(flagDebug)
	Run()
}
