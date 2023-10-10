package main

import (
	"AlgoTN/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

var inputDataChan = make(chan string)

func StartREST() {
	router := gin.Default()

	router.PUT(common.InputEndpoint, receiveInput)
	router.PUT(common.InitEndPoint, initTests)
	router.GET(common.PingEndPoint, ping)

	err := router.Run(":" + strconv.FormatInt(common.RestPort, 10))
	if err != nil {
		return
	}
}

func receiveInput(c *gin.Context) {
	var input common.VmInput
	err := c.BindJSON(&input)

	if err != nil {
		return
	}

	//fmt.Println("Received input request with data", input)

	inputDataChan <- input.Input

	c.Status(http.StatusOK)
}

func ping(c *gin.Context) {
	c.Status(http.StatusOK)
}

func initTests(c *gin.Context) {
	var vmInit common.VmInit
	err := c.BindJSON(&vmInit)

	if err != nil {
		return
	}

	log.Println("Received init request with data", vmInit)

	// Compile program
	compileAndContinue(vmInit)

	c.Status(http.StatusOK)
}

func compileAndContinue(vmInit common.VmInit) {
	log.Println("Compiling program")
	switch vmInit.ProgramType {
	case common.JAVA:
		// javac
		// FIXME: execute with java -cp and specify a default main class or overwrite it
		cmd := exec.Command("javac", "$(find /root/"+vmInit.UserProgram+" -name \"*.java\")", "-d", "/root/"+strings.Split(vmInit.UserProgram, ".")[0])
		cmd.Dir = "/root"
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error compiling Java program:", err)
		}
	case common.CPP:
		cmd := exec.Command("g++", "/root/"+vmInit.UserProgram, "-o", "/root/"+strings.Split(vmInit.UserProgram, ".")[0])
		cmd.Dir = "/root"
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error compiling C++ program:", err)
		}
	case common.PYTHON:
		// No compilation needed
		//exec.Command("mv", "/root/"+vmInit.UserProgram, "/home/container/program/"+vmInit.UserProgram)
	case common.C:
		cmd := exec.Command("gcc", "/root/"+vmInit.UserProgram, "-o", "/root/"+strings.Split(vmInit.UserProgram, ".")[0])
		cmd.Dir = "/root"
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error compiling C program:", err)
		}
	case common.GOLANG:
		cmd := exec.Command("go", "build", "/root/"+vmInit.UserProgram)
		cmd.Dir = "/root"
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error compiling Go program:", err)
		}
	}

	go startTests(vmInit)
}

func startTests(vmInit common.VmInit) {
	for i := 0; i < vmInit.TestCount; i++ {
		input := <-inputDataChan

		// Run test
		switch vmInit.ProgramType {
		case common.JAVA:
			// java
			if true {
				//if input.Type == common.TestTypeInputOutput {
				log.Println("Test type is input/output for Java")

				cmd := exec.Command("java", "-jar", vmInit.UserProgram)
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"
				inputData := input

				//fmt.Println("Input data is", inputData)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					fmt.Println("Error getting stdin pipe:", err)
				}
				pipe, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Println("Error getting stdout pipe:", err)
				}
				err = cmd.Start()
				if err != nil {
					fmt.Println("Error starting command:", err)
				}
				_, err = stdin.Write([]byte(inputData))
				if err != nil {
					fmt.Println("Error writing data to stdin:", err)
				}
				err = stdin.Close()
				if err != nil {
					fmt.Println("Error closing stdin pipe:", err)
				}
				output, err := io.ReadAll(pipe)
				if err != nil {
					fmt.Println("Error reading stdout pipe:", err)
				}
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for the command to exit:", err)
					output = []byte("")
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output

				cmd.Process.Kill()
			}
		case common.CPP:
			// cpp
			if true {
				//if input.Type == common.TestTypeInputOutput {
				log.Println("Test type is input/output for C++")

				cmd := exec.Command("/root/" + strings.Split(vmInit.UserProgram, ".")[0])
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"
				inputData := input

				//fmt.Println("Input data is", inputData)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					fmt.Println("Error getting stdin pipe:", err)
				}
				pipe, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Println("Error getting stdout pipe:", err)
				}
				err = cmd.Start()
				if err != nil {
					fmt.Println("Error starting command:", err)
				}
				_, err = stdin.Write([]byte(inputData))
				if err != nil {
					fmt.Println("Error writing data to stdin:", err)
				}
				err = stdin.Close()
				if err != nil {
					fmt.Println("Error closing stdin pipe:", err)
				}
				output, err := io.ReadAll(pipe)
				if err != nil {
					fmt.Println("Error reading stdout pipe:", err)
				}
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for the command to exit:", err)
					output = []byte("")
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output

				cmd.Process.Kill()
			}
		case common.PYTHON:
			// python
			if true {
				//if input.Type == common.TestTypeInputOutput {
				log.Println("Test type is input/output for Python")

				cmd := exec.Command("python", vmInit.UserProgram) // let's assume it isn't a folder for now
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"
				inputData := input

				bytes := []byte(inputData)

				//fmt.Println("Input data is", inputData)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					fmt.Println("Error getting stdin pipe:", err)
				}
				pipe, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Println("Error getting stdout pipe:", err)
				}
				err = cmd.Start()
				if err != nil {
					fmt.Println("Error starting command:", err)
				}
				_, err = stdin.Write(bytes)
				if err != nil {
					fmt.Println("Error writing data to stdin:", err)
				}
				err = stdin.Close()
				if err != nil {
					fmt.Println("Error closing stdin pipe:", err)
				}
				log.Println("Input written to stdin")
				output, err := io.ReadAll(pipe)
				if err != nil {
					fmt.Println("Error reading stdout pipe:", err)
				}
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for the command to exit:", err)
					output = []byte("")
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output

				cmd.Process.Kill()
			}
		case common.C:
			// c
			if true {
				//if input.Type == common.TestTypeInputOutput {
				log.Println("Test type is input/output for C")

				cmd := exec.Command("/root/" + strings.Split(vmInit.UserProgram, ".")[0])
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"

				bytes := []byte(input)

				//fmt.Println("Input data is", inputData)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					fmt.Println("Error getting stdin pipe:", err)
				}
				pipe, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Println("Error getting stdout pipe:", err)
				}
				err = cmd.Start()
				if err != nil {
					fmt.Println("Error starting command:", err)
				}
				_, err = stdin.Write(bytes)
				if err != nil {
					fmt.Println("Error writing data to stdin:", err)
				}
				err = stdin.Close()
				if err != nil {
					fmt.Println("Error closing stdin pipe:", err)
				}
				log.Println("Input written to stdin")
				output, err := io.ReadAll(pipe)
				if err != nil {
					fmt.Println("Error reading stdout pipe:", err)
				}
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for the command to exit:", err)
					output = []byte("")
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output

				cmd.Process.Kill()
			}
		case common.GOLANG:
			// golang
			if true {
				//if input.Type == common.TestTypeInputOutput {
				log.Println("Test type is input/output for Golang")

				cmd := exec.Command("/root/" + strings.Split(vmInit.UserProgram, ".")[0])
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"
				inputData := input

				//fmt.Println("Input data is", inputData)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					fmt.Println("Error getting stdin pipe:", err)
				}
				pipe, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Println("Error getting stdout pipe:", err)
				}
				err = cmd.Start()
				if err != nil {
					fmt.Println("Error starting command:", err)
				}
				_, err = stdin.Write([]byte(inputData))
				if err != nil {
					fmt.Println("Error writing data to stdin:", err)
				}
				err = stdin.Close()
				if err != nil {
					fmt.Println("Error closing stdin pipe:", err)
				}
				output, err := io.ReadAll(pipe)
				if err != nil {
					fmt.Println("Error reading stdout pipe:", err)
				}
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for the command to exit:", err)
					output = []byte("")
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output

				cmd.Process.Kill()
			}
		}
	}
}
