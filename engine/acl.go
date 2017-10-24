package engine

import (
	"errors"
	"strings"
	"github.com/nicholaskh/golib/set"
)

var (
	skipAclCmd set.Set
)

func InitAclEnv(){
	skipAclCmd = set.NewSet()
	skipAclCmd.Add(CMD_VIDO_CHAT)
	skipAclCmd.Add(CMD_PING)
	skipAclCmd.Add(CMD_AUTH_CLIENT)
	skipAclCmd.Add(CMD_AUTH_SERVER)
	skipAclCmd.Add(CMD_SUBS)
	skipAclCmd.Add(CMD_UNSUBS)
	skipAclCmd.Add(CMD_TOKEN)
	skipAclCmd.Add(CMD_APPKEY)
	skipAclCmd.Add(CMD_DISOLVE)
	skipAclCmd.Add(CMD_CREATE_USER)
	skipAclCmd.Add(CMD_ADD_USER_INTO_ROOM)
	skipAclCmd.Add(CMD_CREATEROOM)
	skipAclCmd.Add(CMD_JOINROOM)
	skipAclCmd.Add(CMD_LEAVEROOM)
	skipAclCmd.Add(CMD_UPDATE_OR_ADD_PUSH_ID)
	skipAclCmd.Add(CMD_SET_OFF_NOTIFY)
}

func AclCheck(cli *Client, cmd string) (err error) {
	if !cli.IsClient() && !cli.IsServer() && !skipAclCmd.Contains(cmd){
		return errors.New("Need Auth first")
	}
	return nil
}

func TokenCheck(cli *Cmdline) error {

	if cli.Cmd == CMD_PING  ||
		cli.Cmd == CMD_VIDO_CHAT ||
		cli.Cmd == CMD_AUTH_CLIENT ||
		cli.Cmd == CMD_AUTH_SERVER ||
		cli.Cmd == CMD_TOKEN ||
		cli.Cmd == CMD_APPKEY ||
		cli.Cmd == CMD_SETUUID {
		return nil
	}

	temp := strings.SplitN(cli.Params, " ", 2)
	if len(temp) == 2{
		cli.Params = temp[1]
	}

	if cli.Client.IsClient() {
		err := checkClientToken(cli.Client, temp[0])
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
