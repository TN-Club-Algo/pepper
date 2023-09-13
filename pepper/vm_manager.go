package main

import (
	"AlgoTN/common"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
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

	ActiveVMs map[string]int
)

func init() {
	vmAddresses = make(map[string]string)
	usedIps = make(map[string]string)

	ActiveVMs = make(map[string]int)
}

func StartVM(codeURL string, request common.TestRequest) {
	ActiveVMs[request.ID] = 2048 // ram in MB
	defer delete(ActiveVMs, request.ID)

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
	b, err := os.ReadFile("/etc/algotn/vm_config.json")
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Determined hostname:", hostDevName)

	kernelBootArgs := "ro console=ttyS0 reboot=k panic=1 pci=off"
	kernelBootArgs += " ip=" + fcAddress + "::" + tapAddress + ":" + maskLong + "::eth0:off"

	config := string(b)
	// new mac fcAddress is the ip fcAddress in hex and 00 00 at the end
	config = strings.Replace(config, "kernelBootArgs", kernelBootArgs, 1)                       // kernel boot args
	config = strings.Replace(config, "AA:BB:CC:DD:EE:FF", ipv4ToHex(fcAddress)+":00:00", 1)     // mac fcAddress
	config = strings.Replace(config, "2048", "2048", 1)                                         // ram
	config = strings.Replace(config, "fc0", hostDevName, 1)                                     // host network name
	config = strings.Replace(config, "/root/rootfs.ext4", "/tmp/rootfs"+hostDevName+".ext4", 1) // disk location
	config = strings.Replace(config, "/root/fc1-disk.ext4", "/tmp/"+hostDevName+".ext4", 1)     // additional disk location

	//defer os.Remove("/root/" + hostDevName + ".ext4")

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Temp config adjusted.")

	// Copy rootfs
	exec.Command("rm", "-f", "/etc/algotn/rootfs"+hostDevName+".ext4")
	err = exec.Command("cp", "/etc/algotn/rootfs.ext4", "/etc/algotn/rootfs"+hostDevName+".ext4").Run()
	if err != nil {
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Copied rootfs.")

	// Share user's program and test program using initrd
	err = createDisk(hostDevName, codeURL, request.Extension)
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

	fcCmd := exec.Command("/etc/algotn/firecracker-bin", "--api-sock", socket, "--config-file", configFile)
	err = fcCmd.Start()
	pid := fcCmd.Process.Pid
	if err != nil {
		fmt.Println("[", hostDevName, "]", "Error starting firecracker VM:", err)
		return
	}

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Firecracker VM started!")

	// Move user's program to container user and change permissions
	key, _ := os.ReadFile("/etc/algotn/id_rsa")
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
		"mkdir /tmp", // fix for golang binary compilation
		"rm -f /root/pepper.out",
		"nohup /root/pepper-vm > /root/pepper.out 2>&1 &",
	}
	command := strings.Join(commands, "; ")

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Running commands...")

	if err := session.Start(command); err != nil {
		fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Error running commands:", err.Error())
	}

	session.Close()

	time.Sleep(500 * time.Millisecond)

	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Firecracker VM ready!")
	vmAddresses[hostDevName] = fcAddress

	startRam, _ := common.CalculateMemory(pid)

	// We are ready for tests
	StartTest(pid, startRam, hostDevName, request)

	// Cleanup
	session, _ = conn.NewSession()
	session.Run("reboot")
	fcCmd.Process.Kill()
	exec.Command("rm", "-f", "/etc/algotn/rootfs"+hostDevName+".ext4").Run()
	exec.Command("rm", "-f", "/etc/algotn/"+hostDevName+".ext4").Run()

	delete(usedIps, hostDevName)
	delete(usedIps, tapHost)
	delete(usedIps, nextHost)
	delete(usedIps, nextHost2)
	delete(vmAddresses, hostDevName)
	fmt.Println("[", hostDevName, time.Now().Format("15:04:05"), "]", "Stopped firecracker VM.")
}

func createDisk(name string, codeURL string, extension string) error {
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
	cmd = exec.Command("cp", "/etc/algotn/pepper-vm", "/tmp/"+name+"/pepper-vm")
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}
	// Download user program
	DownloadAndSaveFile(WebsiteAddress+codeURL, "/tmp/"+name, extension) /*
		cmd = exec.Command("curl", WebsiteAddress+codeURL, "-O", "-H", "x-auth-secret-key "+Secret, "--output-dir", "/tmp/"+name)
		err = cmd.Run()
		if err != nil {
			fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
			return err
		}*/
	cmd = exec.Command("umount", "/tmp/"+name)
	err = cmd.Run()
	if err != nil {
		fmt.Println("[", name, time.Now().Format("15:04:05"), "]", err)
		return err
	}

	return nil
}

func StartTest(pid int, startMemory int, vmID string, testRequest common.TestRequest) {
	_, ok := vmAddresses[vmID]
	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Starting test for VM", vmID, "at", vmAddresses[vmID])
	if ok {
		problemInfo, _ := FetchProblemInfo(WebsiteAddress + testRequest.InfoURL)
		testCount := len(problemInfo.Tests)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Problem info received:", problemInfo)

		// test purpose
		data := common.VmInit{
			ProgramType: testRequest.Language,
			UserProgram: testRequest.UserProgram,
			IsDirectory: false, // should be dynamic
			TestCount:   testCount,
		}
		problemSlug := testRequest.ProblemSlug
		b, _ := json.Marshal(data)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Sending init request to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

		// wait 100ms
		time.Sleep(100 * time.Millisecond)

		var request, err = http.NewRequest("PUT", "http://"+vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InitEndPoint, strings.NewReader(string(b)))
		if err != nil {
			fmt.Println(err)
			go sendTestResult(testRequest.ID, testRequest.ProblemSlug, "Test not reachable", false)
			return
		}

		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		secondTimeout := 3 * time.Second
		if testRequest.Language == common.GOLANG {
			secondTimeout = 10 * time.Second
		}
		client := &http.Client{
			Timeout: secondTimeout,
		}
		response, err := client.Do(request)
		if err != nil {
			go sendTestResult(testRequest.ID, testRequest.ProblemSlug, "Compilation or VM error", false)
			return
		}
		defer client.CloseIdleConnections()
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				return
			}
		}(response.Body)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Init request sent to VM", vmID, "at", vmAddresses[vmID])

		// TODO: Wait for the VM to confirm, if compile has failed, then send test results

		for i := 0; i < testCount; i++ {
			passed, testResponse, timeTaken, finalMemoryUsage := SendInput(i, pid, vmID, problemInfo.Tests[i].Type,
				problemInfo.Tests[i].InputURL, problemInfo.Tests[i].OutputURL, testRequest.TimeLimit)

			finalMemoryUsage -= startMemory
			if !passed {
				fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test failed for VM", vmID, "at", vmAddresses[vmID])
				go sendInnerTestResult(testRequest.ID, i, problemSlug, testResponse, timeTaken, finalMemoryUsage, true, false)
				return
			} else {
				go sendInnerTestResult(testRequest.ID, i, problemSlug, "Test passed", timeTaken, finalMemoryUsage, i == (testCount-1), true)
			}
		}
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "All tests passed for VM", vmID, "at", vmAddresses[vmID])
	}
}

// SendInput Returns if the test passed, the response, the time taken and the final memory usage
func SendInput(pid int, p int, vmID string, testType string, inputURL string, outputURL string, timeLimit int) (bool, string, int, int) {
	pbTimeout := time.Duration(timeLimit) * time.Second
	input, _ := DownloadAsText(WebsiteAddress + inputURL)
	output, _ := DownloadAsText(WebsiteAddress + outputURL)
	var structInput = common.VmInput{ID: vmID, Input: input, Type: testType}

	var b, _ = json.Marshal(structInput)
	//fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Sending input to VM", vmID, "at", vmAddresses[vmID], "with data", string(b))

	var request, err = http.NewRequest("PUT", "http://"+vmAddresses[vmID]+":"+strconv.FormatInt(common.RestPort, 10)+common.InputEndpoint, strings.NewReader(string(b)))
	if err != nil {
		return false, "Error sending input", 0, 0
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return false, "Error sending input", 0, 0
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(response.Body)
	start := time.Now().UnixMilli()

	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Input sent to VM", vmID, "at", vmAddresses[vmID])
	fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Waiting for result from VM", vmID, "at", vmAddresses[vmID])

	// Wait for the result on the websocket for max 1 second
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
		return false, "Fatal error", -1, -1
	}

	timeout := pbTimeout * 3
	c.SetReadDeadline(time.Now().Add(timeout))

	// TODO: remove memory used by the VM

	receiveType, rsp, err := c.ReadMessage()
	duration := time.Now().UnixMilli() - start
	if err != nil {
		// is it a timeout?
		if err, ok := err.(net.Error); ok && err.Timeout() {
			fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Timeout on VM", vmID, "at", vmAddresses[vmID])
			memory, _ := common.CalculateMemory(pid)
			return false, "Timeout", int(timeout.Milliseconds()), memory
		}
		if duration > pbTimeout.Milliseconds() {
			fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Timeout on VM", vmID, "at", vmAddresses[vmID])
			memory, _ := common.CalculateMemory(pid)
			return false, "Timeout", int(duration), memory
		}
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "ReadMessage failed:", err)
		memory, _ := common.CalculateMemory(pid)
		return false, "Fatal error", int(duration), memory
	}
	if duration > pbTimeout.Milliseconds() {
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Time limit exceeded", vmID, "at", vmAddresses[vmID])
		memory, _ := common.CalculateMemory(pid)
		return false, "Time limit exceeded", int(duration), memory
	}
	if receiveType != websocket.TextMessage {
		fmt.Printf("received type(%d) != websocket.TextMessage(%d)\n", receiveType, websocket.TextMessage)
		memory, _ := common.CalculateMemory(pid)
		return false, "Fatal error", int(duration), memory
	}

	// remove the last \n and unuseful spaces
	rspStr := strings.TrimSpace(string(rsp))
	outputFix := strings.TrimSpace(output)

	//fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Received output:", rspStr, "expected:", output)
	if rspStr == outputFix || string(rsp) == output || common.NormalizeLineEndings(rspStr) == common.NormalizeLineEndings(outputFix) {
		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test passed!")
		memory, _ := common.CalculateMemory(pid)
		return true, "", int(duration), memory
	} else {
		// write the output to a file
		os.WriteFile("output.txt", []byte(rspStr), 0644)
		// expected
		os.WriteFile("expected.txt", []byte(outputFix), 0644)

		fmt.Println("[", vmID, time.Now().Format("15:04:05"), "]", "Test failed!")
		memory, _ := common.CalculateMemory(pid)
		return false, "Wrong answer on test " + strconv.Itoa(p), int(duration), memory
	}
}
