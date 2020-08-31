package misc

import (
	"fmt"
	"os"
)

func CheckErr(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(-1)
	}
}
