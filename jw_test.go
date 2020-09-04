package main

import (
	"fmt"
	"testing"

	"github.com/sodapanda/junkwire/datastructure"
)

func TestFsm(t *testing.T) {
	fsm := datastructure.NewFsm("closed")
	fsm.AddRule("closed", "doOpen", "opend", func() {
		fmt.Println("close to open")
	})

	fsm.AddRule("opend", "doClose", "closed", func() {
		fmt.Println("open to close")
	})

	fsm.OnEvent("doOpen")
	fsm.OnEvent("doClose")
}
