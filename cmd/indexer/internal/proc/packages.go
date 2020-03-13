package proc

import (
	"fmt"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/obj"
)

type PackStatus int

const (
	PackStateDisable PackStatus = iota // статус пакета - активировать
	PackStateEnable                    // статус пакета - заблокировать
)

// SetPackStatus активирует или блокирует пакет для индексации
func SetPackStatus(r *obj.Repo, status PackStatus, packs []string) error {
	var err error
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
			fmt.Println("проиндексируйте пакеты командой index [...pacnames]")
			fmt.Println("выгрузите данные в индекс-файл командой populate")
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
			r.CleanPacks()
			fmt.Println("выгрузите данные в индекс-файл командой populate")
		}
	}
	if err != nil {
		panic(fmt.Errorf(":SetPackStatus:%v", err))
	}
	return nil
}
