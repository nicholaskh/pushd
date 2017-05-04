package engine

import (
	"errors"
)

func AclCheck(cli *Client, cmd string) (err error) {
	if cmd != CMD_AUTH_CLIENT && cmd != CMD_AUTH_SERVER && cmd != CMD_PING &&
		cmd != CMD_TOKEN && cmd != CMD_APPKEY && !cli.IsClient() && !cli.IsServer() {
		return errors.New("Need Auth first")
	}
	return nil
}
