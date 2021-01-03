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
func Init(logfile string) {
	f, err := os.OpenFile(logfile,
		os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err.Error())
	}

	logger = log.New(f, "", log.LstdFlags)
}

//PLog print long
func PLog(msg string) {
	logger.Println(msg)
}
