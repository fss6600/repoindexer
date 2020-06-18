package obj

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/pmshoot/repoindexer/cmd/indexer/internal/utils"
)

const errExecFileMsg = ":ExecFile:"

// todo использовать r.FilesPackRepo() -
func searchExecFile(root string, regExp *regexp.Regexp) ([]string, error) {
	execFilesList := []string{}
	fpCh := make(chan string) // channel for filepath
	erCh := make(chan error)  // channel for error

	go utils.DirWalk(root, fpCh, erCh)

	for {
		select {
		case err := <-erCh:
			return nil, err
		case fp, ok := <-fpCh:
			if ok { // канал еще не закрыт
				if regExp.MatchString(fp) {
					execFilesList = append(execFilesList, fp)
				}
			} else {
				return execFilesList, nil
			}
		}
	}
}

func selectExecFileByUser(fList []string) string {
	scanner := bufio.NewScanner(os.Stdin)
	count := len(fList)
	for {
		fmt.Printf("введите число от 1 до %d:\n", count)
		for i := 0; i < count; i++ {
			fmt.Printf("\t[%d]: '%v'\n", i+1, fList[i])
		}
		scanner.Scan()
		choice := scanner.Text()
		choice_int, err := strconv.Atoi(choice)
		if err == nil && choice_int != 0 && choice_int <= count {
			return fList[choice_int-1]
		}
	}
}

func defineExecFile(r *Repo, pack string) string {
	var (
		execFilesList []string
		execFile      string
	)
	packRoot := filepath.Join(r.Path(), pack)
	execRegEx, _ := regexp.Compile("^.+\\.exe$")

	if execFilesList, err = searchExecFile(packRoot, execRegEx); err != nil {
		panic(fmt.Errorf("%v:%v:%v", errExecFileMsg, "set", err))
	}

	switch len(execFilesList) {
	case 0:
		execFile = "noexec"
	case 1:
		execFile = execFilesList[0]
	default:
		fmt.Printf("Выберите исполняемый файл для пакета '%v'\n", pack)
		execFile = selectExecFileByUser(execFilesList)
	}

	return execFile
}

func ShowEmptyExecFiles(r *Repo) {
	emptyList := r.EmptyExecFilesList()
	if len(emptyList) > 0 {
		fmt.Println("\n\tДля следующих пакетов требуется указать исполняемый файл:")
		for _, pack := range emptyList {
			fmt.Printf("\t\t%v\n", pack)
		}
		fmt.Println("\tЗапустите программу с командой 'exec check'")
	}
}
