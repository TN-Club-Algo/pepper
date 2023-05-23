package main

import "fmt"

func main() {
	//Connect("127.0.0.1", "")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err != nil {
			fmt.Println(cmd)
			if cmd == "startvm" {
				StartVM("/root/test-vm")
			}
		}
	}
}
