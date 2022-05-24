package messaging

import (
	"github.com/glide-im/glideim/im/client"
	"github.com/glide-im/glideim/im/group"
	"github.com/glide-im/glideim/im/message"
	"github.com/glide-im/glideim/pkg/logger"
)

type Interface func(from int64, device int64, msg *message.Message) error

var handler Interface = handleMessage

func HandleMessage(from int64, device int64, msg *message.Message) error {
	return handler(from, device, msg)
}

func SetInterfaceImpl(i Interface) {
	handler = i
}

func dispatchGroupMessage(gid int64, msg *message.ChatMessage) error {
	return group.DispatchMessage(gid, msg)
}

func dispatchRecallMessage(gid int64, msg *message.ChatMessage) error {
	return group.DispatchRecallMessage(gid, msg)
}

func enqueueMessage(uid int64, message *message.Message) {
	err := client.EnqueueMessage(uid, message)
	if err != nil {
		logger.E("%v", err)
	}
}

func enqueueMessage2Device(uid int64, device int64, message *message.Message) {
	err := client.EnqueueMessageToDevice(uid, device, message)
	if err != nil {
		logger.E("%v", err)
	}
}
