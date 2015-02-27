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

func processReq(input string) (string, error) {
	cl := resolveCmd(input)
	return processCmd(cl)
}

// TODO protocal
func resolveCmd(input string) *cmdline {
	parts := strings.Split(input, " ")
	return &cmdline{cmd: strings.TrimRight(parts[0], string([]rune{0, 13, 10})), params: parts[1:]}
}

func processCmd(cl *cmdline) (string, error) {
	// TODO log debug
	fmt.Println("cmd: " + cl.cmd)
	var ret string
	switch cl.cmd {
	case "subscribe":
		ret = subscribe(cl.params[0])

	case "publish":
		return "def", nil

	}
	return "", errors.New("Cmd not found: " + cl.cmd)
}
