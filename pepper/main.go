package main

import (
	"fmt"
	"os"
)

var (
	MaxRam         = 16384
	WebsiteAddress = "https://algo.limpsword.fr"
	Secret         = "secret"
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
