package main

// TODO subscribe count of channel
func subscribe(cli *client, channel string) string {
	_, exists := cli.channels[channel]
	if exists {
		return CMD_ALREADY_SUBSCRIBED
	} else {
		cli.channels[channel] = 1
		clients, exists := pubsubChannels[channel]
		if exists {
			clients = append(clients, cli)
		} else {
			clients = []*client{cli}
		}
		pubsubChannels[channel] = clients
		return CMD_SUBSCRIBED
	}
}
