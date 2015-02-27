package main

func subscribe(cli *client, channel string) string {
	clients, exists := pubsubChannels[channel]
	if exists {
		clients = append(clients, cli)
	} else {
		clients = []*client{cli}
	}
	pubsubChannels[channel] = clients
	return CMD_SUBSCRIBED
}
