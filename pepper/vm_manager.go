package main

import (
	"AlgoTN/common"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	baseIp          string = "10.0.0.1"
	baseHostDevName string = "10001"
)

var (
	vmAddresses    map[string]string
	usedIps        map[string]string
	justStartedVMs []string
)

func init() {
	vmAddresses = make(map[string]string)
	usedIps = make(map[string]string)
}

func StartVM(folder string, request common.TestRequest) {
	// Find an available IP
	//maskLong := "255.255.255.0"
	maskLong := "255.255.255.252"
	maskShort := "/30"
	//maskShort := "/24"

	fcAddress := GetAvailableIP(baseHostDevName, baseIp)
	hostDevName := strings.ReplaceAll(fcAddress, ".", "")
	usedIps[hostDevName] = fcAddress

	tapAddress := GetAvailableIP(baseHostDevName, baseIp)
	tapHost := strings.ReplaceAll(tapAddress, ".", "")
	usedIps[tapHost] = tapAddress

	nextIp := GetAvailableIP(baseHostDevName, baseIp)
	nextHost := strings.ReplaceAll(nextIp, ".", "")
	usedIps[nextHost] = nextIp

	nextIp2 := GetAvailableIP(baseHostDevName, baseIp)
	nextHost2 := strings.ReplaceAll(nextIp2, ".", "")
	usedIps[nextHost2] = nextIp2

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Found fcAddress:", fcAddress)

	// Edit config
	b, err := os.ReadFile("/root/vm_config.json")
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Determined hostname:", hostDevName)

	kernelBootArgs := "ro console=ttyS0 reboot=k panic=1 pci=off"
	kernelBootArgs += " ip=" + fcAddress + "::" + tapAddress + ":" + maskLong + "::eth0:off"

	config := string(b)
	// new mac fcAddress is the ip fcAddress in hex and 00 00 at the end
	config = strings.Replace(config, "kernelBootArgs", kernelBootArgs, 1)                        // kernel boot args
	config = strings.Replace(config, "AA:BB:CC:DD:EE:FF", ipv4ToHex(fcAddress)+":00:00", 1)      // mac fcAddress
	config = strings.Replace(config, "2048", "2048", 1)                                          // ram
	config = strings.Replace(config, "fc0", hostDevName, 1)                                      // host network name
	config = strings.Replace(config, "/root/rootfs.ext4", "/root/rootfs"+hostDevName+".ext4", 1) // disk location
	config = strings.Replace(config, "/root/fc1-disk.ext4", "/root/"+hostDevName+".ext4", 1)     // additional disk location

	//defer os.Remove("/root/" + hostDevName + ".ext4")

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Temp config adjusted.")

	// Copy rootfs
	exec.Command("rm", "-f", "/root/rootfs"+hostDevName+".ext4")
	err = exec.Command("cp", "/root/rootfs.ext4", "/root/rootfs"+hostDevName+".ext4").Run()
	if err != nil {
		return
	}

	// Share user's program and test program using initrd
	err = createDisk(hostDevName, folder)
	if err != nil {
		return
	}
	//exec.Command("cd root/" + folder + " ; find . -print0 | cpio --null --create --verbose --format=newc > " + hostDevName + ".cpio")

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Temp disk created with user program and pepper-vm.")

	// Create firecracker VM config
	configFile := "temp_vm_config_" + hostDevName + ".json"
	err = os.WriteFile(configFile, []byte(config), 0600)
	//defer os.Remove(configFile)
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Starting Firecracker VM...")

	// Start firecracker VM
	socket := "/tmp/firecracker" + hostDevName + ".socket"
	// Remove socket if it exists
	err = exec.Command("rm", "-f", socket).Run()
	if err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Error removing socket:", err)
		return
	}

	// Set host network
	exec.Command("ip", "link", "del", hostDevName).Run()
	err = exec.Command("ip", "tuntap", "add", "dev", hostDevName, "mode", "tap").Run()
	if err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Error creating host network:", err)
		return
	}
	exec.Command("sysctl", "-w", "net.ipv4.conf."+hostDevName+".proxy_arp=1").Run()
	exec.Command("sysctl", "-w", "net.ipv6.conf."+hostDevName+".disable_ipv6=1").Run()
	err = exec.Command("ip", "addr", "add", tapAddress+maskShort, "dev", hostDevName).Run()
	if err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Error adding ip address:", err)
		return
	}
	err = exec.Command("ip", "link", "set", "dev", hostDevName, "up").Run()
	if err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Error setting host network up:", err)
		return
	}

	justStartedVMs = append(justStartedVMs, hostDevName)
	// remove after 12s
	go func() {
		time.Sleep(12 * time.Second)
		for i := range justStartedVMs {
			if justStartedVMs[i] == hostDevName {
				justStartedVMs = append(justStartedVMs[:i], justStartedVMs[i+1:]...)
				break
			}
		}
	}()

	fcCmd := exec.Command("/root/firecracker-bin", "--api-sock", socket, "--config-file", configFile)
	err = fcCmd.Start()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error starting firecracker VM:", err)
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Firecracker VM started!")

	// Move user's program to container user and change permissions
	key, _ := os.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)

	var conn *ssh.Client

	if err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "(PVKEY) Error creating ssh connection:", err)
		return
	}

	maxAttempts := 10
	attempt := 1
	for attempt <= maxAttempts {
		time.Sleep(1 * time.Second) // Wait for a few seconds before attempting SSH connection

		conn, err = ssh.Dial("tcp", fcAddress+":22", &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		})

		if err == nil {
			defer conn.Close()
			break // SSH connection successful, break out of the loop
		}

		attempt++
	}

	if attempt > maxAttempts {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "(ATTEMPTS) Error creating ssh connection:", err)
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "SSH connection successful!")

	session, _ := conn.NewSession()

	// mount the needed files (user program and pepper) then run pepper-vm
	// Create a single command that is semicolon seperated
	commands := []string{
		"mkdir -p /mnt",
		"mount /dev/vdb /mnt",
		"mv /mnt/* /root/ 2>/dev/null",
		"mkdir -p /home/container",
		"mv /root/program /home/container/ 2>/dev/null",
		"chown -R container /home/container 2>/dev/null",
		"chgrp -R container /home/container 2>/dev/null",
		"chmod -R 500 /home/container 2>/dev/null",
		"mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2", // fix for pepper-vm binary execution
		"nohup /root/pepper-vm > pepper.out 2>&1 &",
	}
	command := strings.Join(commands, "; ")

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Running commands...")

	if err := session.Start(command); err != nil {
		panic("Failed to run command: " + command + "\nBecause: " + err.Error())
	}

	session.Close()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Firecracker VM ready!")
	vmAddresses[hostDevName] = fcAddress

	// We are ready for tests
	StartTest(hostDevName, request)

	// Cleanup
	session, _ = conn.NewSession()
	session.Run("reboot")
	fcCmd.Process.Kill()
	exec.Command("rm", "-f", "/root/rootfs"+hostDevName+".ext4").Run()
	exec.Command("rm", "-f", "/root/"+hostDevName+".ext4").Run()

	delete(usedIps, hostDevName)
	delete(usedIps, tapHost)
	delete(usedIps, nextHost)
	delete(usedIps, nextHost2)
	delete(vmAddresses, hostDevName)
	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Stopped firecracker VM.")
}

func createDisk(name string, folder string) error {
	cmd := exec.Command("dd", "if=/dev/zero", "of="+name+".ext4", "bs=1M", "count=20")
	err := cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("mkfs.ext4", name+".ext4")
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("rm", "-rf", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("mkdir", "-p", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("mount", name+".ext4", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("cp", "-r", folder+"/.", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	cmd = exec.Command("umount", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}

	/*filesToMove := []string{"/root/pepper-vm"}
	for _, fileToMove := range filesToMove {
		srcPath := fileToMove
		destPath := filepath.Join(programPath, fileToMove)

		if err := os.Rename(srcPath, destPath); err != nil {
			return err
		}
	}*/

	return nil
}

func StartTest(vmID string, testRequest common.TestRequest) {
	_, ok := vmAddresses[vmID]
	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Starting test for VM", vmID, "at", vmAddresses[vmID])
	if ok {
		// test purpose
		data := common.VmInit{
			ProgramType: common.PYTHON, // should be dynamic
			UserProgram: testRequest.UserProgram,
			IsDirectory: false, // should be dynamic
			TestType:    testRequest.TestType,
			TestCount:   testRequest.TestCount,
		}
		b, _ := json.Marshal(data)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Sending init request to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

		// wait 100ms
		time.Sleep(100 * time.Millisecond)

		var request, err = http.NewRequest("PUT", "http://"+vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InitEndPoint, strings.NewReader(string(b)))
		if err != nil {
			panic(err)
		}

		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		client := &http.Client{
			Timeout: 2 * time.Second,
		}
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		defer client.CloseIdleConnections()
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				return
			}
		}(response.Body)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Init request sent to VM", vmID, "at", vmAddresses[vmID])

		testJson := testRequest.Tests
		test := common.InnerInputOutputTest{}
		err = json.Unmarshal([]byte(testJson), &test)
		if err != nil {
			panic(err)
		}
		for i := range test.Inputs {
			if !SendInput(vmID, test.Inputs[i], test.Outputs[i]) {
				fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test failed for VM", vmID, "at", vmAddresses[vmID])
				go sendInnerTestResult(testRequest.ID, i, false)
				go sendTestResult(testRequest.ID, false)
				return
			} else {
				go sendInnerTestResult(testRequest.ID, i, true)
			}
		}
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "All tests passed for VM", vmID, "at", vmAddresses[vmID])
		go sendTestResult(testRequest.ID, true)
	}
}

func SendInput(vmID string, input string, expectedOutput string) bool {
	var structInput = common.VmInput{ID: vmID, Input: input}

	var b, _ = json.Marshal(structInput)
	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Sending input to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

	var request, err = http.NewRequest("PUT", "http://"+vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InputEndpoint, strings.NewReader(string(b)))
	if err != nil {
		panic(err)
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(response.Body)

	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Input sent to VM", vmID, "at", vmAddresses[vmID])
	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Waiting for result from VM", vmID, "at", vmAddresses[vmID])

	// Wait for the result on the websocket
	u := url.URL{Scheme: "ws", Host: vmAddresses[vmID] + ":8888", Path: "/ws"}
	c, res, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		bReason, _ := io.ReadAll(res.Body)
		log.Fatalf("dial: %v, reason: %v\n", err, string(bReason))
	}
	defer c.Close()
	err = c.WriteMessage(websocket.TextMessage, []byte("output"))
	if err != nil {
		log.Fatalf("write: %v", err)
		return false
	}

	receiveType, rsp, err := c.ReadMessage()
	if err != nil {
		log.Println("[", vmID, time.Now().Format("15:04:05"), "]", "ReadMessage failed:", err)
		return false
	}
	if receiveType != websocket.TextMessage {
		log.Printf("received type(%d) != websocket.TextMessage(%d)\n", receiveType, websocket.TextMessage)
		return false
	}

	// remove the last \n and unuseful spaces
	rspStr := strings.Trim(string(rsp), "\n")
	rspStr = strings.Trim(rspStr, " ")

	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Received output:", rspStr, "expected:", expectedOutput)
	if rspStr == expectedOutput {
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test passed!")
		return true
	} else {
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test failed!")
		return false
	}
}
