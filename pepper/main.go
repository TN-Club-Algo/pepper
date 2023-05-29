package main

import (
	"AlgoTN/common"
	"fmt"
	"os"
	"strings"
)

func main() {
	Connect("127.0.0.1", "")

	// Tick
	go tick()

	fmt.Println("Pepper initialized, type 'help' for a list of commands.")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err == nil {
			if strings.HasPrefix(cmd, "startvm") {
				split := strings.Split(cmd, " ")
				if len(split) != 2 {
					fmt.Println("Starting test VM...")
					StartVM("/root/test-vm", common.TestRequest{
						TestType:    common.TestTypeInputOutput,
						Tests:       `{"input": "1 2 3 4 5", "output": "1 2 3 4 5"}`,
						UserProgram: "/root/test-vm",
						TestCount:   1,
						ID:          "iamanid",
					})
					continue
				}
				folder := split[1]
				fmt.Println("Starting VM with folder", folder, "...")
				StartVM(folder, common.TestRequest{})
			} else if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
