package main

import (
	"AlgoTN/common"
	"fmt"
	"os"
	"strings"
)

var (
	MaxRam = 16384
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
			if strings.HasPrefix(cmd, "startvm") {
				split := strings.Split(cmd, " ")
				if len(split) != 2 {
					fmt.Println("Starting test VM...")
					tests := common.InnerInputOutputTest{
						Inputs:  []string{"1 2 3 4 5", "je suis un pain"},
						Outputs: []string{"1 2 3 4 5", "je suis un pain"},
					}
					//bytes, _ := json.Marshal(tests)

					go StartVM("/root/test-vm", common.TestRequest{
						ProblemName: "test",
						TestType:    common.TestTypeInputOutput,
						Tests:       tests,
						UserProgram: "program.py",
						TestCount:   2,
						ID:          "iamanid",
					})
					continue
				}
				folder := split[1]
				fmt.Println("Starting VM with folder", folder, "...")
				go StartVM(folder, common.TestRequest{})
			} else if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
