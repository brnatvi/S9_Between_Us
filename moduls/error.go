package moduls

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
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
		fmt.Printf("%s: %s%v%s\n", msg, string(colorPurple), err, string(colorReset))
	}
}

func HandleFatalError(err error, msg string) {
	if err != nil {
		fmt.Printf("%s: %s%v%s\n", msg, string(colorRed), err, string(colorReset))
	}
}

func PanicMessage(msg string) {
	fmt.Printf("%s%s%s\n", string(colorPurple), msg, string(colorReset))
}

func UnexpectedMessage(msg string) {
	fmt.Printf("%s%s%s\n", string(colorYellow), msg, string(colorReset))
}

func PrintError(msg string) {
	fmt.Printf("%s%s%s\n", string(colorRed), msg, string(colorReset))
}

func DebugPrint(msg interface{}) {
	if debug {
		fmt.Printf("%s%q%s\n", string(colorGreen), msg, string(colorReset))
	}
}

func CheckTypeEquality(wantedType byte, recieved []byte) int {
	if recieved[POS_TYPE:POS_LENGTH][0] != wantedType {
		len := binary.BigEndian.Uint16(recieved[POS_LENGTH:POS_HASH])
		log.Printf("%s Not a %d was recieved, but %d. Message: %s %s\n\n",
			string(colorYellow),
			wantedType,
			recieved[POS_TYPE:POS_LENGTH][0],
			hex.EncodeToString(recieved[POS_HASH:(POS_HASH+len)]),
			string(colorReset))
		return -1
	}
	return 0
}

func NoDatumRecieved() error {
	return errors.New("NO_DATUM was received")
}
