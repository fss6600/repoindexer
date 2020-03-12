package proc

import (
	"fmt"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

// SetPackStatus активирует или блокирует пакет для индексации
func SetPackStatus(repo *obj.Repo, setDisable bool, pack []string) error {
	if setDisable {
		fmt.Printf("деактивация пакетов: %s\n", pack)
	} else {
		fmt.Printf("активация пакетов: %s\n", pack)
	}

	return nil
}

