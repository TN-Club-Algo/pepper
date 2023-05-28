package main

import (
	"AlgoTN/common"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os/exec"
	"strconv"
)

var inputDataChan = make(chan []byte)

func StartREST() {
	router := gin.Default()

	router.PUT(common.InputEndpoint, receiveInput)
	router.PUT(common.InitEndPoint, initTests)

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

	inputDataChan <- []byte(input.Input)

	c.Status(http.StatusOK)
}

func initTests(c *gin.Context) {
	testCount, _ := strconv.Atoi(c.Request.URL.Query().Get("testCount"))
	vmInit := common.VmInit{
		ProgramType: c.Request.URL.Query().Get("programType"),
		UserProgram: c.Request.URL.Query().Get("userProgram"),
		IsDirectory: c.Request.URL.Query().Get("isDirectory") == "true",
		TestType:    c.Request.URL.Query().Get("testType"),
		TestCount:   testCount,
	}

	// Compile program
	go compileAndContinue(vmInit)

	c.Status(http.StatusOK)
}

func compileAndContinue(vmInit common.VmInit) {
	switch vmInit.ProgramType {
	case common.JAVA:
		// javac
		exec.Command("javac", "$(find /home/container/program -name \"*.java\")")
	case common.CPP:
	case common.PYTHON:
		// No compilation needed
	case common.C:

	}

	go startTests(vmInit)
}

func startTests(vmInit common.VmInit) {
	switch vmInit.TestType {
	case common.TestTypeInputOutput:

	}

	for i := 0; i < vmInit.TestCount; i++ {
		// Run test
		switch vmInit.ProgramType {
		case common.JAVA:

		case common.CPP:
		case common.PYTHON:
			// python
			if vmInit.TestType == common.TestTypeInputOutput {
				cmd := exec.Command("python", vmInit.UserProgram) // let's assume it isn't a folder for now
				inputData := <-inputDataChan
				stdin, _ := cmd.StdinPipe()
				_ = cmd.Start()
				_, _ = stdin.Write(inputData)
				stdin.Close()
				pipe, _ := cmd.StdoutPipe()
				output, _ := io.ReadAll(pipe)
				_ = cmd.Wait()

				// write output to file
				exec.Command("touch", "/home/container/output.txt").Run()
				exec.Command("echo", string(output), ">>", "/home/container/output.txt").Run()
			}
		case common.C:
		}
	}
}
