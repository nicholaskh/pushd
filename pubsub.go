package main

import (
	"fmt"
)

// TODO subscribe count of channel
func subscribe(cli *client, channel string) string {
	_, exists := cli.channels[channel]
	if exists {
		return fmt.Sprintf("%s %s", OUTPUT_ALREADY_SUBSCRIBED, channel)
	} else {
		cli.channels[channel] = 1
		clients, exists := pubsubChannels[channel]
		if exists {
			clients[cli] = 1
		} else {
			clients = map[*client]int{cli: 1}
		}
		pubsubChannels[channel] = clients
		return fmt.Sprintf("%s %s", OUTPUT_SUBSCRIBED, channel)
	}
}

func unsubscribe(cli *client, channel string) string {
	_, exists := cli.channels[channel]
	if exists {
		delete(cli.channels, channel)
		delete(pubsubChannels[channel], cli)
		return fmt.Sprintf("%s %s", OUTPUT_UNSUBSCRIBED, channel)
	} else {
		return fmt.Sprintf("%s %s", OUTPUT_NOT_SUBSCRIBED, channel)
	}
}

func publish(channel, msg string) string {
	clients, exists := pubsubChannels[channel]
	if exists {
		msgByte := []byte(msg)
		// TODO log debug
		//fmt.Println(clients)
		for cli, _ := range clients {
			go cli.conn.Write(msgByte)
		}
	}
	return OUTPUT_PUBLISHED
}
