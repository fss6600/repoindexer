package proc

var err error

// general
const (
	fnReglament = "__REGLAMENT__"
	errMsg      = "Error:"
	doPopMsg    = "\n\tВыгрузите данные в индекс-файл командой 'pop'\n"
	doIndexMsg  = "\n\tПроиндексируйте пакеты командой 'index [...pacnames]'\n"
	noChangeMsg = "Изменений нет\n"
)

// list
const (
	catBlock     = "блок"
	catActive    = ""
	catNoIndexed = "!инд"
	tmplListOut  = "[%4v] %v\n"
)

// packages
const (
	PackStateDisable PackStatus = iota // статус пакета - активировать
	PackStateEnable                    // статус пакета - заблокировать
)

// reglament
const (
	reglOnMessage  string = "Режим регламента активирован [on]"
	reglOffMessage string = "Режим регламента деактивирован [off]"
)

// status
const (
	timeLayout = "2006-01-02 15:04:05"
)
