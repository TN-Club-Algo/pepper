package main

import (
	"AlgoTN/common"
	"github.com/gin-gonic/gin"
	"net/http"
)

func StartREST() {
	router := gin.Default()

	router.PUT(common.InputEndpoint, receiveInput)
	router.PUT(common.InitEndPoint, initTests)

	err := router.Run("localhost:" + string(common.RestPort))
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

	// Handle input

	c.IndentedJSON(http.StatusOK, nil)
}

func initTests(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, nil)
}
