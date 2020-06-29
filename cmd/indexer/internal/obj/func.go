package obj

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func searchExecFile(root string, regExp *regexp.Regexp) []string {
	execFilesList := []string{}
	for fInfo := range dirWalk(root) {
		if regExp.MatchString(fInfo.Path) {
			execFilesList = append(execFilesList, fInfo.Path)
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
				return fmt.Errorf("не найден пакет: %q\n", fp)
			}
			if info.IsDir() { // skip directory
				return nil
			}
			// fPath, _ := filepath.Rel(root, fp) // trim base Path repopath/packname
			fInfo.Path = fp
			fInfo.Size = info.Size()
			fInfo.MDate = info.ModTime().UnixNano()
			fInfoCh <- fInfo
			return nil
		})
		if err != nil {
			panic(fmt.Errorf("dirWalk: %v", err))
		}
		close(fInfoCh)
	}()

	return fInfoCh
}

// func dirWalk(root string, fpCh chan<- *FileInfo, erCh chan<- error) {
// 	err := filepath.Walk(root, func(fp string, info os.FileInfo, er error) error {
// 		if er != nil {
// 			return fmt.Errorf("не найден пакет: %q\n", fp)
// 		}
// 		if info.IsDir() { // skip directory
// 			return nil
// 		}
// 		// fPath, _ := filepath.Rel(root, fp) // trim base Path repopath/packname
// 		fInfo := &FileInfo{
// 			Path:  fp,
// 			Size:  info.Size(),
// 			MDate: info.ModTime().UnixNano(),
// 		}
// 		fpCh <- fInfo
// 		return nil
// 	})
// 	if err != nil {
// 		erCh <- err
// 		return
// 	}
// 	close(fpCh)
// }
