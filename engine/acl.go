package engine

import (
	"errors"
)

func AclCheck(cli *Client, cmd string) (err error) {
	if cmd != CMD_AUTH_CLIENT && cmd != CMD_AUTH_SERVER && !cli.Authed {
		return errors.New("Need Auth first")
	}
	return nil
}
