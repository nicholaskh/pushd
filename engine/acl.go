package engine

import (
	"errors"
	"strings"
)

func AclCheck(cli *Client, cmd string) (err error) {
	if !cli.IsClient() &&
		cmd != CMD_PING &&
		cmd != CMD_AUTH_CLIENT &&
		cmd != CMD_AUTH_SERVER &&
		cmd != CMD_SUBS &&
		cmd != CMD_UNSUBS &&
		cmd != CMD_TOKEN &&
		cmd != CMD_APPKEY &&
		cmd != CMD_DISOLVE &&
		!cli.IsServer() {
		return errors.New("Need Auth first")
	}

	return nil
}

func TokenCheck(cli *Cmdline) error {

	if cli.Cmd == CMD_PING  || cli.Cmd == CMD_AUTH_CLIENT || cli.Cmd == CMD_AUTH_SERVER ||
		cli.Cmd == CMD_TOKEN || cli.Cmd == CMD_APPKEY || cli.Cmd == CMD_SETUUID{
		return nil
	}

	temp := strings.SplitN(cli.Params, " ", 2)
	if len(temp) == 2{
		cli.Params = temp[1]
	}

	if cli.Client.IsClient() {
		err := checkClientToken(cli.Client.uuid, temp[0])
		if err != nil {
			cli.Client.ClearIdentity()
			return err
		}

	} else {
		if !checkServerToken(temp[0]){
			return errors.New("token error")
		}
	}

	return nil
}
