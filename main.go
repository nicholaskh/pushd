package main

var (
	pubsubChannels map[string]map[*client]int = make(map[string]map[*client]int)
)

func main() {
	startServ()
}
