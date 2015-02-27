package main

var (
	pubsubChannels map[string][]*client = make(map[string][]*client)
)

func main() {
	startServ()
}
