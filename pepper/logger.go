package main

import (
	"io"
	"log"
	"os"
)

var (
	logFile *os.File
)

func init() {
	err := os.MkdirAll("/var/log/pepper", 0755)
	if err != nil {
		panic(err)
	}

	logFile, _ = os.OpenFile("/var/log/pepper/latest.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
