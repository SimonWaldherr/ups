package main

import (
	"log"
	"os"
	"simonwaldherr.de/go/ups"
)

func main() {
	ups.LogInit(os.Stdout, os.Stderr, os.Stderr)

	ups.Hub.Init()

	log.SetFlags(log.Ltime | log.Lshortfile)

	ups.Labels, ups.Ltemplate = ups.ParseLabels("labels")

	ups.Printer = ups.LoadPrinter("drucker.csv")

	go ups.PrintMessages()
	go ups.InitTelnet()
	ups.InitHTTP()
}
