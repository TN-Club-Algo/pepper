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
	baseIp string = "10.0.0.1"
)

var (
	vmAddresses    map[string]string
	usedIps        []string
	justStartedVMs []string
)

func init() {
	vmAddresses = make(map[string]string)
	usedIps = make([]string, 20)
}

func StartVM(folder string) {
	// Find an available IP
	maskLong := "255.255.255.252"
	maskShort := "/30"

	fcAddress := GetAvailableIP(baseIp, usedIps)
	usedIps = append(usedIps, fcAddress)
	tapAddress := GetAvailableIP(baseIp, usedIps)
	usedIps = append(usedIps, tapAddress)

	hostDevName := strings.Replace(fcAddress, ".", "", -1)

	fmt.Println("[", hostDevName, "]", "Found fcAddress:", fcAddress)

	// Edit config
	b, err := os.ReadFile("/root/vm_config.json")
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, "]", "Determined hostname:", hostDevName)

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

	fmt.Println("[", hostDevName, "]", "Temp config adjusted.")

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

	fmt.Println("[", hostDevName, "]", "Temp disk created with user program and pepper-vm.")

	// Create firecracker VM config
	configFile := "temp_vm_config_" + hostDevName + ".json"
	err = os.WriteFile(configFile, []byte(config), 0600)
	//defer os.Remove(configFile)
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, "]", "Starting Firecracker VM...")

	// Start firecracker VM
	socket := "/tmp/firecracker" + hostDevName + ".socket"
	// Remove socket if it exists
	err = exec.Command("rm", "-f", socket).Run()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error removing socket:", err)
		return
	}

	// Set host network
	exec.Command("ip", "link", "del", hostDevName).Run()
	err = exec.Command("ip", "tuntap", "add", "dev", hostDevName, "mode", "tap").Run()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error creating host network:", err)
		return
	}
	exec.Command("sysctl", "-w", "net.ipv4.conf."+hostDevName+".proxy_arp=1").Run()
	exec.Command("sysctl", "-w", "net.ipv6.conf."+hostDevName+".disable_ipv6=1").Run()
	err = exec.Command("ip", "addr", "add", tapAddress+maskShort, "dev", hostDevName).Run()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error adding ip address:", err)
		return
	}
	err = exec.Command("ip", "link", "set", "dev", hostDevName, "up").Run()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error setting host network up:", err)
		return
	}

	justStartedVMs = append(justStartedVMs, hostDevName)
	// remove after 7s
	go func() {
		time.Sleep(7 * time.Second)
		for i := range justStartedVMs {
			if justStartedVMs[i] == hostDevName {
				justStartedVMs = append(justStartedVMs[:i], justStartedVMs[i+1:]...)
				break
			}
		}
	}()

	err = exec.Command("/root/firecracker-bin", "--api-sock", socket, "--config-file", configFile).Start()
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error starting firecracker VM:", err)
		return
	}

	fmt.Println("[", hostDevName, "]", "Firecracker VM started!")

	// Move user's program to container user and change permissions
	key, _ := os.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)

	var conn *ssh.Client

	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error creating ssh connection:", err)
		return
	}

	maxAttempts := 5
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
			break // SSH connection successful, break out of the loop
		}

		attempt++
	}

	if attempt > maxAttempts {
		fmt.Println("[", hostDevName, "]", "Error creating ssh connection:", err)
		return
	}

	fmt.Println("[", hostDevName, "]", "SSH connection successful!")

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

	fmt.Println("[", hostDevName, "]", "Running commands...")

	if err := session.Start(command); err != nil {
		panic("Failed to run command: " + command + "\nBecause: " + err.Error())
	}

	session.Close()
	conn.Close()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("[", hostDevName, "]", "Firecracker VM ready!")
	vmAddresses[hostDevName] = fcAddress

	// We are ready for tests
	StartTest(hostDevName)
}

func createDisk(name string, folder string) error {
	cmd := exec.Command("dd", "if=/dev/zero", "of="+name+".ext4", "bs=1M", "count=20")
	err := cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("mkfs.ext4", name+".ext4")
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("rm", "-rf", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("mkdir", "-p", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("mount", name+".ext4", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("cp", "-r", folder+"/.", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
		return err
	}
	cmd = exec.Command("umount", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, "]", err)
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

func StartTest(vmID string) {
	_, ok := vmAddresses[vmID]
	fmt.Println("[", vmID, "]", "Starting test for VM", vmID, "at", vmAddresses[vmID])
	if ok {
		// test purpose
		data := common.VmInit{
			ProgramType: common.PYTHON,
			UserProgram: "program.py",
			IsDirectory: false,
			TestType:    common.TestTypeInputOutput,
			TestCount:   1,
		}
		b, _ := json.Marshal(data)

		fmt.Println("[", vmID, "]", "Sending init request to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

		var request, err = http.NewRequest("PUT", "http://"+vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InitEndPoint, strings.NewReader(string(b)))
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

		fmt.Println("[", vmID, "]", "Init request sent to VM", vmID, "at", vmAddresses[vmID])

		SendInput(vmID, "abc", "abc")
	}
}

func SendInput(vmID string, input string, expectedOutput string) {
	var structInput = common.VmInput{ID: vmID, Input: input}

	var b, _ = json.Marshal(structInput)
	fmt.Println("[", vmID, "]", "Sending input to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

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

	fmt.Println("[", vmID, "]", "Input sent to VM", vmID, "at", vmAddresses[vmID])
	fmt.Println("[", vmID, "]", "Waiting for result from VM", vmID, "at", vmAddresses[vmID])

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
		return
	}

	receiveType, rsp, err := c.ReadMessage()
	if err != nil {
		log.Println("[", vmID, "]", "ReadMessage failed:", err)
		return
	}
	if receiveType != websocket.TextMessage {
		log.Printf("received type(%d) != websocket.TextMessage(%d)\n", receiveType, websocket.TextMessage)
		return
	}

	// remove the last \n and unuseful spaces
	rspStr := strings.Trim(string(rsp), "\n")
	rspStr = strings.Trim(rspStr, " ")

	log.Println("[", vmID, "]", "Received output:", rspStr, "expected:", expectedOutput)
	if rspStr == expectedOutput {
		fmt.Println("[", vmID, "]", "Test passed!")
	} else {
		fmt.Println("[", vmID, "]", "Test failed!")
	}
}

func EndVM() {
	// Send reboot command, wait 1s, kill process if it exists and delete socket
}
