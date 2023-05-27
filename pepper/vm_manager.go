package main

import (
	"AlgoTN/common"
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net/http"
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
	vmAddresses map[string]string
	usedIps     []string
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

	fmt.Println("Found fcAddress:", fcAddress)

	// Edit config
	b, err := os.ReadFile("/root/vm_config.json")
	if err != nil {
		return
	}

	hostDevName := strings.Replace(fcAddress, ".", "", -1)

	fmt.Println("Determined hostname:", hostDevName)

	kernelBootArgs := "ro console=ttyS0 reboot=k panic=1 pci=off"
	kernelBootArgs += " ip=" + fcAddress + "::" + tapAddress + ":" + maskLong + "::eth0:off"

	config := string(b)
	// new mac fcAddress is the ip fcAddress in hex and 00 00 at the end
	config = strings.Replace(config, "kernelBootArgs", kernelBootArgs, 1)                    // kernel boot args
	config = strings.Replace(config, "AA:BB:CC:DD:EE:FF", ipv4ToHex(fcAddress)+":00:00", 1)  // mac fcAddress
	config = strings.Replace(config, "2048", "2048", 1)                                      // ram
	config = strings.Replace(config, "fc0", hostDevName, 1)                                  // host network name
	config = strings.Replace(config, "/root/fc1-disk.ext4", "/root/"+hostDevName+".ext4", 1) // initrd location

	//defer os.Remove("/root/" + hostDevName + ".ext4")

	fmt.Println("Temp config adjusted.")

	// Share user's program and test program using initrd
	err = createDisk(hostDevName, folder)
	if err != nil {
		return
	}
	//exec.Command("cd root/" + folder + " ; find . -print0 | cpio --null --create --verbose --format=newc > " + hostDevName + ".cpio")

	fmt.Println("Temp disk created with user program and pepper-vm.")

	// Create firecracker VM config
	configFile := "temp_vm_config_" + hostDevName + ".json"
	err = os.WriteFile(configFile, []byte(config), 0600)
	//defer os.Remove(configFile)
	if err != nil {
		return
	}

	fmt.Println("Starting Firecracker VM...")

	// Start firecracker VM
	socket := "/tmp/firecracker" + hostDevName + ".socket"
	// Remove socket if it exists
	err = exec.Command("rm", "-f", socket).Run()
	if err != nil {
		fmt.Println("Error removing socket:", err)
		return
	}

	// Set host network
	exec.Command("ip", "link", "del", hostDevName).Run()
	err = exec.Command("ip", "tuntap", "add", "dev", hostDevName, "mode", "tap").Run()
	if err != nil {
		fmt.Println("Error creating host network:", err)
		return
	}
	exec.Command("sysctl", "-w", "net.ipv4.conf."+hostDevName+".proxy_arp=1").Run()
	exec.Command("sysctl", "-w", "net.ipv6.conf."+hostDevName+".disable_ipv6=1").Run()
	err = exec.Command("ip", "addr", "add", tapAddress+maskShort, "dev", hostDevName).Run()
	if err != nil {
		fmt.Println("Error adding ip address:", err)
		return
	}
	err = exec.Command("ip", "link", "set", "dev", hostDevName, "up").Run()
	if err != nil {
		fmt.Println("Error setting host network up:", err)
		return
	}

	err = exec.Command("/root/firecracker-bin", "--api-sock", socket, "--config-file", configFile).Start()
	if err != nil {
		fmt.Println("Error starting firecracker VM:", err)
		return
	}

	fmt.Println("Firecracker VM started!")

	// Move user's program to container user and change permissions
	key, _ := os.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)

	var conn *ssh.Client

	if err != nil {
		fmt.Println("Error creating ssh connection:", err)
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
		fmt.Println("Error creating ssh connection:", err)
		return
	}

	// we'll need to wait for the VM to be ready
	session, _ := conn.NewSession()
	session.Run("mv /root/program /home/container/program")
	session.Run("chown -R container /home/container")
	session.Run("chgrp -R container /home/container")
	session.Run("chmod -R 500 /home/container") // execute and read

	// mount the needed files (user program and pepper)
	session.Run("mkdir -p /mnt")
	session.Run("mount /dev/vdb /mnt")

	// Start pepper-vm
	session.Run("./pepper-vm")

	session.Close()
	conn.Close()

	fmt.Println("Firecracker VM ready!")
	vmAddresses[hostDevName] = fcAddress

	// We are ready for tests, listen to the results
	StartTest(hostDevName)
}

func createDisk(name string, folder string) error {
	cmd := exec.Command("dd", "if=/dev/zero", "of="+name+".ext4", "bs=1M", "count=20")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("mkfs.ext4", name+".ext4")
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("mkdir", "-p", "/tmp/"+name)
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("mount", name+".ext4", "/tmp/"+name)
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("cp", "-a", folder+"/.", "/tmp/"+name)
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("umount", "/tmp/"+name)
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)

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
	fmt.Println("Starting test for VM", vmID, "at", vmAddresses[vmID])
	if ok {
		var request, err = http.NewRequest("POST", vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InitEndPoint, nil)
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
	}
}

func SendInput(vmID string, input string) {
	var structInput = common.VmInput{ID: vmID, Input: input}

	var result, _ = json.Marshal(structInput)
	var request, _ = http.NewRequest("POST", vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InputEndpoint, bytes.NewBuffer(result))

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
}

func EndVM() {
	// Send reboot command, wait 1s, kill process if it exists and delete socket
}
