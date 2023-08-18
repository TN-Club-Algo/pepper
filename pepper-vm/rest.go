package main

import (
	"AlgoTN/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os/exec"
	"strconv"
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

	fmt.Println("Received init request with data", vmInit)

	// Compile program
	compileAndContinue(vmInit)

	c.Status(http.StatusOK)
}

func compileAndContinue(vmInit common.VmInit) {
	switch vmInit.ProgramType {
	case common.JAVA:
		// javac
		exec.Command("javac", "$(find /home/container/program -name \"*.java\")")
	case common.CPP:
		exec.Command("g++", "/root/"+vmInit.UserProgram, "-o", "/root/program")
	case common.PYTHON:
		// No compilation needed
		//exec.Command("mv", "/root/"+vmInit.UserProgram, "/home/container/program/"+vmInit.UserProgram)
	case common.C:
		exec.Command("gcc", "/root/"+vmInit.UserProgram, "-o", "/root/program")
	}

	go startTests(vmInit)
}

func startTests(vmInit common.VmInit) {
	for i := 0; i < vmInit.TestCount; i++ {
		input := <-inputDataChan

		// Run test
		switch vmInit.ProgramType {
		case common.JAVA:

		case common.CPP:
		case common.PYTHON:
			// python
			if true {
				//if input.Type == common.TestTypeInputOutput {
				fmt.Println("Test type is input/output for Python")

				cmd := exec.Command("python", vmInit.UserProgram) // let's assume it isn't a folder for now
				cmd.Dir = "/root"
				//cmd.Dir = "/home/container/program"
				inputData := input

				fmt.Println("Input data is", inputData)

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
				}

				//fmt.Println("Output is", string(output))

				// write output to the channel which will send it to the client
				outputChan <- output
			}
		case common.C:
		}
	}
}
