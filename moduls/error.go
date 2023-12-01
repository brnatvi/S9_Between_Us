package moduls

import (
	"fmt"
	"log"
)

// debug variable, set true to enable debug fmt.prints
const debug = true

// misc funcs
func HandlePanicError(err error, msg string) {
	if err != nil {
		log.Panic(msg)
	}
}

func HandleFatalError(err error, msg string) {
	if err != nil {
		log.Fatal(msg)
	}
}

func DebugPrint(msg interface{}) {
	if debug {
		fmt.Printf("%q \n", msg)
	}
}
