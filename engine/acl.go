package engine

import (
	"errors"
	"time"
	"strings"
)

func AclCheck(cli *Client, cmd string) (err error) {
	if cmd != CMD_AUTH_CLIENT && cmd != CMD_AUTH_SERVER && cmd != CMD_PING &&
		cmd != CMD_TOKEN && cmd != CMD_APPKEY && !cli.IsClient() && !cli.IsServer() {
		return errors.New("Need Auth first")
	}

	return nil
}

func TokenCheck(cli *Cmdline) error {

	if cli.Cmd == CMD_PING  || cli.Cmd == CMD_AUTH_CLIENT || cli.Cmd == CMD_AUTH_SERVER ||
		cli.Cmd == CMD_TOKEN || cli.Cmd == CMD_APPKEY {
		return nil
	}

	temp := strings.SplitN(cli.Params, " ", 2)
	if len(temp) == 2{
		cli.Params = temp[1]
	}

	if  cli.token == "" || cli.token != temp[0] {
		cli.Client.ClearIdentity()
		return errors.New("Token Illegal")
	}
	if time.Now().Unix() - cli.lastTimestamp > 7200 {
		cli.Client.ClearIdentity()
		return errors.New("Token expired")
	}
	if cli.IsClient(){
		cli.lastTimestamp = time.Now().Unix()
	}

	return nil
}
