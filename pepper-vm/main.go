package main

func main() {
	go StartREST()
	go StartWebSocketServer()

	select {}
}
