package handler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// SetReglamentMode обрабатывает команду `regl`
// активирует/деактивирует режим регламента репозитория
func SetReglamentMode(repoPath, mode string) error {
	const (
		reglOnMessage  string = "Режим регламента активирован [on]"
		reglOffMessage string = "Режим регламента деактивирован [off]"
	)
	// проверка на наличие файла-флага, определение режима регламента
	fRegl := filepath.Join(repoPath, fnReglament)
	modeOn := reglIsSet(repoPath)

	switch mode {
	// вывод режима регламента
	case "":
		if modeOn {
			fmt.Println(reglOnMessage)
		} else {
			fmt.Println(reglOffMessage)
		}
		// активация режима регламента
	case "on":
		// регламент уже активирован - вывод сообщения и информации кто активировал
		if modeOn {
			owner, _ := ioutil.ReadFile(fRegl)
			fmt.Println(reglOnMessage, string(owner))
			// активация регламента с записью информации кто активировал
		} else {
			err = ioutil.WriteFile(fRegl, TaskOwnerInfo(), 0644)
			if err != nil {
				return &internalError{
					Text:   "ошибка установки режима регламента",
					Caller: "Reglament::writeFile",
					Err:    err,
				}
			}
			fmt.Println(reglOnMessage)
		}
	// деактивация режима регламента
	case "off":
		// регламент активирован - удаляем файл
		if modeOn {
			if err = os.Remove(fRegl); err != nil {
				return &internalError{
					Text:   "ошибка снятия режима регламента",
					Caller: "Reglament::removeFile",
					Err:    err,
				}
			}
		}
		fmt.Println(reglOffMessage)
	default:
		return &internalError{
			Text:   fmt.Sprintf("неверный режим регламента: %s", mode),
			Caller: "Reglament",
		}
	}
	return nil
}

func reglIsSet(repo string) bool {
	reglf := filepath.Join(repo, fnReglament)
	return FileExists(reglf)
}

func checkRegl(repoPath string) error {
	if !reglIsSet(repoPath) {
		return &internalError{
			Text:   "не установлен режим регламента",
			Caller: "Reglament::checkRegl",
		}
	}
	return nil
}
