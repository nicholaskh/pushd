package engine

import (
	"errors"
)

var (
	ErrNotPermit error = errors.New("Not Permit")
)

// 数据库集合msg_log中各记录的类型
const (
	// 正常消息
	MESSAGE_TYPE_NORMAL = 0

	// 某人创建群聊
	CREATE_CHAT_ROOM =2;

	// 某人撤回消息
	RETRACT_MESSAGE = 3;

	// 某人加入群聊
	SOME_ONE_JOIN_CHAT_ROOM = 4;

	// 某人退出群聊
	SOME_ONE_QUIT_CHAT_ROOM = 5;

	// 某人被踢出群聊
	SOME_ONE_EXPELLED_FROM_CHAT_ROOM = 6;

	// 某人修改了群名称
	SOME_ONE_MODIFY_NAME_OF_CHAT_ROOM = 7;

	// 某人被邀请加入群聊
	SOME_ONE_BE_INVITED_OT_CHAT_ROOM = 8;

	// 群聊解散
	SOME_ONE_DISMISS_CHAT_ROOM = 9;

)