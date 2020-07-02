package handler

import (
	"fmt"
)

// SetPackStatus активирует или блокирует пакет для индексации
func SetPackStatus(r *Repo, status int, packs []string) error {
	var done bool
	switch status {
	// активация пакетов
	case PackStatusActive:
		for _, pack := range packs {
			if !r.PackIsBlocked(pack) {
				fmt.Printf("[ %v ] уже в актуальном состоянии\n", pack)
				continue
			}
			if err = r.EnablePack(pack); err != nil {
				return err
			}
			done = true
		}
		if done {
			fmt.Print(doIndexMsg)
			fmt.Println(doPopMsg)
		}
	// блокирование пакетов
	case PackStatusBlocked:
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
			if err = r.CleanPacks(); err != nil {
				return err
			}
			fmt.Println(doPopMsg)
		}
	}
	return nil
}
