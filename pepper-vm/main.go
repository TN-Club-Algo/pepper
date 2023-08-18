package main

import "os/exec"

func main() {
	exec.Command("rm", "-f", "output.txt").Run()
	exec.Command("touch", "output.txt").Run()

	go StartREST()
	go StartWebSocketServer()

	select {}
}
