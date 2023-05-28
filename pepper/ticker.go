package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func tick() {
	// List all sockets and check if they are reachable
	folder := "/tmp"
	ticker := time.NewTicker(5 * time.Second)

	for {
		<-ticker.C

		files, err := os.ReadDir(folder)
		if err != nil {
			return
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "firecracker") {
				vmIp, _ := strings.CutPrefix(file.Name(), "firecracker")
				vmIp, _ = strings.CutSuffix(vmIp, ".socket")

				// if it is inside justStartedVMs, skip
				justStarted := false
				for _, vm := range justStartedVMs {
					if vm == vmIp {
						justStarted = true
						break
					}
				}
				if justStarted {
					continue
				}

				// Check if the VM is reachable
				resultChan := make(chan bool)
				go checkAPIReachability("http://"+vmIp+":8080/ping", resultChan)

				select {
				case result := <-resultChan:
					if !result {
						removeVM(vmIp)
					}
				}

			}
		}

		ticker.Reset(5 * time.Second)
	}
}

func removeVM(vmIP string) {
	// Remove process associated with the socket
	exec.Command("pkill -9 -f firecracker" + vmIP)

	socket := "/tmp/firecracker" + vmIP + ".socket"
	os.Remove(socket)

	// Remove IP
	exec.Command("ip addr del " + vmIP)

	fmt.Println("Removed VM with IP " + vmIP)
}

func checkAPIReachability(apiURL string, resultChan chan<- bool) {
	// Send a GET request to the API URL
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		resultChan <- false // API is not reachable
		return
	}
	defer resp.Body.Close()

	resultChan <- true // API is reachable
}
