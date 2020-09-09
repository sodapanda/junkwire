package misc

import (
	"fmt"
	"log"
	"os"
)

//CheckErr check error
func CheckErr(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(-1)
	}
}

var logger *log.Logger

//Init init
func Init() {
	f, err := os.OpenFile("junkwire.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err.Error())
	}

	logger = log.New(f, "junkwire", log.LstdFlags)
}

//PLog print long
func PLog(msg string) {
	logger.Println(msg)
}
