package main

import (
	"fmt"
	"os"
)

func main() {
	//Connect("127.0.0.1", "")
	fmt.Println("Pepper initialized, type 'help' for a list of commands.")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err == nil {
			if cmd == "startvm" {
				fmt.Println("Starting VM...")
				StartVM("/root/test-vm")
			} else if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
