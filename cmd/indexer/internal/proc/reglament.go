package proc

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const fnReglament string = "__REGLAMENT__"

// SetReglamentMode активирует/деактивирует режим регламента репозитория
func SetReglamentMode(repoPath, mode string) {
	const (
		reglOnMessage  string = "режим регламента активирован [on]"
		reglOffMessage string = "режим регламента деактивирован [off]"
	)
	fRegl := filepath.Join(repoPath, fnReglament)
	// проверка на наличие файла-флага, определение режима реглавмета
	modeOn := ReglIsSet(fRegl)

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
			// авктивация реглавмента с записью информации кто активировал
		} else {
			if err := ioutil.WriteFile(fRegl, utils.TaskOwnerInfo(), 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Println(reglOnMessage)
		}
	// деактивация режима реглавмента
	case "off":
		// регламент активирован - удаляем файл
		if modeOn {
			if err := os.Remove(fRegl); err != nil {
				log.Fatalln(err)
			}
		}
		fmt.Println(reglOffMessage)
	default:
		fmt.Printf("неверный режим регламента: %s", mode)
	}
}

func ReglIsSet(reglf string) bool {
	return utils.FileExists(reglf)
}

func CheckRegl(repoPath string) {
	fRegl := filepath.Join(repoPath, fnReglament)
	if !ReglIsSet(fRegl) {
		panic("не установлен режим регламента!")
	}
}
