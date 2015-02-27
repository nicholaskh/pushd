package main

import (
	"errors"
	"fmt"
	"strings"
)

type cmdline struct {
	cmd    string
	params []string
}

const (
	CMD_SUBSCRIB    = "subscribe"
	CMD_PUBLISH     = "publish"
	CMD_UNSUBSCRIBE = "unsubscribe"

	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
)

func processReq(cli *client) (string, error) {
	return processCmd(cli, resolveCmd(cli.input))
}

// TODO protocal
func resolveCmd(input string) *cmdline {
	input = trim(input)
	parts := strings.Split(input, " ")
	return &cmdline{cmd: trim(parts[0]), params: parts[1:]}
}

func processCmd(cli *client, cl *cmdline) (string, error) {
	// TODO log debug
	fmt.Println("cmd: " + cl.cmd)
	var ret string
	switch cl.cmd {
	case CMD_SUBSCRIB:
		ret = subscribe(cli, cl.params[0])

	case CMD_PUBLISH:
		if len(cl.params) < 2 {
			// TODO log warning
			return "", errors.New("Publish without msg")
		} else {
			ret = publish(cl.params[0], cl.params[1])
		}

	case CMD_UNSUBSCRIBE:
		ret = unsubscribe(cli, cl.params[0])

	default:
		return "", errors.New("Cmd not found: " + cl.cmd)

	}

	return ret, nil
}

func trim(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
