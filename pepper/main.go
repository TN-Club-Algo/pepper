package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	//Connect("127.0.0.1", "")
	fmt.Println("Pepper initialized, type 'help' for a list of commands.")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err == nil {
			if strings.HasPrefix(cmd, "startvm") {
				split := strings.Split(cmd, " ")
				if len(split) != 2 {
					StartVM("/root/test-vm")
					continue
				}
				folder := split[1]
				fmt.Println("Starting VM with folder", folder, "...")
				StartVM(folder)
			} else if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
