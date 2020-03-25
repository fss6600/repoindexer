package proc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const fnReglament string = "__REGLAMENT__"

// SetReglamentMode активирует/деактивирует режим регламента репозитория
func SetReglamentMode(repoPath, mode string) {
	const tmplErrMsg = "error::packages::SetReglamentMode:"
	const (
		reglOnMessage  string = "режим регламента активирован [on]"
		reglOffMessage string = "режим регламента деактивирован [off]"
	)
	// проверка на наличие файла-флага, определение режима реглавмета
	fRegl := filepath.Join(repoPath, fnReglament)
	modeOn := ReglIsSet(repoPath)

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
			err := ioutil.WriteFile(fRegl, utils.TaskOwnerInfo(), 0644)
			utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
			fmt.Println(reglOnMessage)
		}
	// деактивация режима реглавмента
	case "off":
		// регламент активирован - удаляем файл
		if modeOn {
			err := os.Remove(fRegl)
			utils.CheckError(fmt.Sprintf("%v", tmplErrMsg), &err)
		}
		fmt.Println(reglOffMessage)
	default:
		panic(fmt.Sprintf("неверный режим регламента: %s", mode))
	}
}

func ReglIsSet(repo string) bool {
	reglf := filepath.Join(repo, fnReglament)
	return utils.FileExists(reglf)
}

func CheckRegl(repoPath string) {
	if !ReglIsSet(repoPath) {
		panic("не установлен режим регламента!")
	}
}
