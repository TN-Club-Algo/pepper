package main

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func tick() {
	// List all sockets and check if they are reachable
	folder := "/tmp"
	files, err := os.ReadDir(folder)

	if err != nil {
		return
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "firecracker") {
			vmIp, _ := strings.CutPrefix(file.Name(), "firecracker")
			vmIp, _ = strings.CutSuffix(vmIp, ".socket")

			// Check if the VM is reachable
			resultChan := make(chan bool)
			go checkAPIReachability("https://"+vmIp+":8080", resultChan)

			select {
			case result := <-resultChan:
				if !result {
					removeVM(vmIp)
				}
			case <-time.After(5 * time.Second):
				removeVM(vmIp)
			}

		}
	}
}

func removeVM(vmIP string) {
	// Remove process associated with the socket
	exec.Command("pkill -9 -f firecracker" + vmIP)

	socket := "/tmp/firecracker" + strings.Replace(vmIP, ".", "-", -1) + ".socket"
	os.Remove(socket)

	// Remove IP
	hostDevName := strings.Replace(vmIP, ":", "", -1)
	exec.Command("ip addr del " + vmIP + "/32 dev " + hostDevName)
	exec.Command("ip link set " + hostDevName + " down")
}

func checkAPIReachability(apiURL string, resultChan chan<- bool) {
	// Send a GET request to the API URL
	resp, err := http.Get(apiURL)
	if err != nil {
		resultChan <- false // API is not reachable
		return
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode == http.StatusOK {
		resultChan <- true // API is reachable
	} else {
		resultChan <- false // API is not reachable
	}
}
