package moduls

import (
	"fmt"
	"log"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[1;31m"
	colorGreen  = "\033[1;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[1;34m"
	colorPurple = "\033[1;35m"
	colorCyan   = "\033[1;36m"
	colorWhite  = "\033[1;37m"
)

// debug variable, set true to enable debug fmt.prints
const debug = true

// misc funcs
func HandlePanicError(err error, msg string) {
	if err != nil {
		log.Panic(msg, " : ", string(colorPurple), err, string(colorReset))
	}
}

func HandleFatalError(err error, msg string) {
	if err != nil {
		log.Fatal(msg, " : ", string(colorRed), err, string(colorReset))
	}
}

func DebugPrint(msg interface{}) {
	if debug {
		fmt.Printf("%s%q%s\n", string(colorGreen), msg, string(colorReset))
	}
}

func checkIfErrorRecieved(wantedType byte, recieved []byte) int {
	if recieved[4:5][0] != wantedType {
		log.Printf("%sNot a %d was recieved, but %d. Error message: %s%s\n\n", string(colorYellow), wantedType, recieved[4:5][0], string(recieved[7:]), string(colorReset))
		return -1
	}
	return 0
}
