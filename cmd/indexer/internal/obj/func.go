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

func searchExecFile(root string, regExp *regexp.Regexp) []string {
	execFilesList := []string{}
	for fInfo := range dirWalk(root) {
		if regExp.MatchString(fInfo.Path) {
			fp, err := filepath.Rel(root, fInfo.Path)
			utils.CheckError("obj.searchExecFile:", &err)
			execFilesList = append(execFilesList, fp)
		}
	}
	return execFilesList
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
		choiceInt, err := strconv.Atoi(choice)
		if err == nil && choiceInt != 0 && choiceInt <= count {
			return fList[choiceInt-1]
		}
	}
}

func defineExecFile(r *Repo, pack string) string {
	var (
		execFilesList []string
		execFile      string
	)
	packRoot := filepath.Join(r.Path(), pack)
	execRegEx, _ := regexp.Compile(`^.+\.exe$`)
	execFilesList = searchExecFile(packRoot, execRegEx)

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

// ShowEmptyExecFiles выводит на консоль список пакетов, для которых требуется указать исполняемый файл
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

// dirWalk Рекурсивно обходит указанную папку и возвращает канал
// с данными о файлах
func dirWalk(root string) chan FileInfo {
	fInfoCh := make(chan FileInfo)
	fInfo := FileInfo{}

	go func() {
		err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
			if er != nil {
				return fmt.Errorf("не найден пакет: %q", filepath.Base(fp))
			}
			if info.IsDir() { // skip directory
				return nil
			}
			fInfo.Path = fp
			fInfo.Size = info.Size()
			fInfo.MDate = info.ModTime().UnixNano()
			fInfoCh <- fInfo
			return nil
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		close(fInfoCh)
	}()

	return fInfoCh
}
