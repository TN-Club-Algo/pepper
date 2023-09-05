package main

import (
	"AlgoTN/common"
	"fmt"
	"os"
	"strconv"
)

var (
	WebsiteAddress = common.GetEnv("WEBSITE_URL", "https://algo.limpsword.fr")
	Secret         = common.GetEnv("API_SECRET", "api_secret")
	MaxRam, _      = strconv.Atoi(common.GetEnv("MAX_RAM", "16384"))
)

func main() {
	Connect("127.0.0.1:6379", "")

	// Tick
	go tick()

	fmt.Println("Pepper initialized, type 'help' for a list of commands.")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err == nil {
			if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
