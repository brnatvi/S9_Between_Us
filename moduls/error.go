package moduls

import (
	"encoding/binary"
	"fmt"
	"log"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[1;31m" // fatal error
	colorGreen  = "\033[1;32m"
	colorYellow = "\033[1;33m" // unexpected message type
	colorBlue   = "\033[1;34m"
	colorPurple = "\033[1;35m" // not fatal error
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

func PanicMessage(msg string) {
	log.Panic(string(colorPurple), msg, string(colorReset))
}

func UnexpectedMessage(msg string) {
	log.Panic(string(colorYellow), msg, string(colorReset))
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

func CheckTypeEquality(wantedType byte, recieved []byte) int {
	if recieved[4:5][0] != wantedType {
		len := binary.BigEndian.Uint16(recieved[5:7])
		log.Printf("%s Not a %d was recieved, but %d. Error message: %v %s\n\n", string(colorYellow), wantedType, recieved[4:5][0], string(recieved[7:(7+len)]), string(colorReset))
		return -1
	}
	return 0
}
