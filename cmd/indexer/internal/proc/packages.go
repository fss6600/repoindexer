package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

type PackStatus int

// SetPackStatus активирует или блокирует пакет для индексации
func SetPackStatus(r *obj.Repo, status PackStatus, packs []string) {
	const errMsg = errMsg + ":packages::SetPackStatus:"
	done := false

	switch status {
	// активация пакетов
	case PackStateEnable:
		for _, pack := range packs {
			if !r.PackIsBlocked(pack) {
				fmt.Printf("[ %v ] уже в актуальном состоянии\n", pack)
				continue
			}
			err = r.EnablePack(pack)
			done = true
		}
		if done {
			fmt.Print(doIndexMsg)
			fmt.Println(doPopMsg)
		}
	// блокирование пакетов
	case PackStateDisable:
		for _, pack := range packs {
			if r.PackIsBlocked(pack) {
				fmt.Printf("[ %v ] уже заблокирован\n", pack)
				continue
			}
			if !r.PackIsActive(pack) {
				fmt.Printf("[ %v ] не найден\n", pack)
				continue
			}
			err = r.DisablePack(pack)
			done = true
		}
		// очистка БД от данных заблокированных пакетов
		if done {
			err = r.CleanPacks()
			utils.CheckError(fmt.Sprintf("%v", errMsg), &err)
			fmt.Println(doPopMsg)
		}
	}
	utils.CheckError(fmt.Sprintf("%v", errMsg), &err)
}
