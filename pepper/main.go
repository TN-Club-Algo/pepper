package main

import (
	"AlgoTN/common"
	"fmt"
	"os"
	"strconv"
)

var (
	MaxRam, _      = strconv.Atoi(common.GetEnv("MAX_RAM", "16384"))
	WebsiteAddress = common.GetEnv("WEBSITE_URL", "https://algo.limpsword.fr")
	Secret         = common.GetEnv("API_SECRET", "api_secret")
	RedisAddress   = common.GetEnv("REDIS_ADDRESS_PORT", "127.0.0.1:6379")
	RedisPassword  = common.GetEnv("REDIS_PASS", "")
)

func main() {
	fmt.Println("Initializing Pepper...")
	Connect(RedisAddress, RedisPassword)

	// Tick
	go tick()

	fmt.Println("Pepper initialized, type 'help' for a list of commands.")
	var cmd string
	for {
		_, err := fmt.Scanln(&cmd)
		if err == nil {
			if cmd == "exit" || cmd == "stop" {
				os.Exit(0)
				return
			} else {
				fmt.Println("Unknown command.")
			}
		}
	}
}
