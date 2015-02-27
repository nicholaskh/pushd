package main

import (
	"fmt"
)

func publish(channel, msg string) string {
	clients, exists := pubsubChannels[channel]
	if exists {
		msgByte := []byte(msg)
		// TODO log debug
		fmt.Println(clients)
		for _, cli := range clients {
			go cli.conn.Write(msgByte)
		}
	}
	return CMD_PUBLISHED
}
