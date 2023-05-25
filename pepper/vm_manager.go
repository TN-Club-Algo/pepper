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
	"strings"
)

const (
	baseIp string = "10.0.0.0"
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
	address := GetAvailableIP(baseIp, usedIps)

	fmt.Println("Found address: ", address)

	// Edit config
	b, err := os.ReadFile("/root/vm_config.json")
	if err != nil {
		return
	}

	hostDevName := strings.Replace(address, ".", "", -1)

	fmt.Println("Determined hostname: ", hostDevName)

	config := string(b)
	// new mac address is the ip address in hex and 00 00 at the end
	config = strings.Replace(config, "AA:BB:CC:DD:EE:FF", ipv4ToHex(address)+":00:00", 1)    // mac address
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
	socket := "/tmp/firecracker" + strings.Replace(address, ".", "-", -1) + ".socket"
	exec.Command("firecracker-bin", "--api-sock", socket, "--config-file", configFile)

	fmt.Println("Firecracker VM started!")

	// Set host network
	exec.Command("ip", "addr", "add", address+"/32", "dev", hostDevName)
	exec.Command("ip", "link", "set", hostDevName, "up")

	// Move user's program to container user and change permissions
	key, _ := os.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)
	conn, err := ssh.Dial("tcp", address+":22", &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	})

	session, _ := conn.NewSession()
	session.Run("mv /root/program /home/container/program")
	session.Run("chown -R container /home/container")
	session.Run("chgrp -R container /home/container")
	session.Run("chmod -R 500 /home/container") // execute and read

	// mount the needed files (user program and pepper)
	session.Run("mkdir /mnt")
	session.Run("mount /dev/vdb /mnt")

	// Start pepper-vm
	session.Run("./pepper-vm")

	session.Close()
	conn.Close()

	fmt.Println("Firecracker VM ready!")

	// We are ready for tests, listen to the results
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
	cmd = exec.Command("cp", "-r", folder, "/tmp/"+name)
	cmd.Run()
	fmt.Println(cmd.Stdout)
	fmt.Println(cmd.Err)
	cmd = exec.Command("sudo umount /tmp/" + name)
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
	if ok {

	}
}

func SendInput(vmID string, input string) {
	var structInput = common.VmInput{ID: vmID, Input: input}

	var result, _ = json.Marshal(structInput)
	var request, _ = http.NewRequest("POST", vmAddresses[vmID]+":"+string(common.RestPort)+common.InputEndpoint, bytes.NewBuffer(result))

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
