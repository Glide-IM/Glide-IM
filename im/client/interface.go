package client

import (
	"github.com/glide-im/glideim/im/message"
	"github.com/glide-im/glideim/pkg/logger"
	"time"
)

type ServerInfo struct {
	Online      int64
	MaxOnline   int64
	MessageSent int64
	StartAt     int64

	OnlineCli []Info
}

type Interface interface {
	ClientSignIn(oldUid int64, uid int64, device int64) error

	ClientLogout(uid int64, device int64) error

	EnqueueMessage(uid int64, device int64, message *message.Message) error
}

type MessageHandler func(from int64, device int64, message *message.Message) error

// messageHandleFunc 所有客户端消息都传递到该函数处理
var messageHandleFunc MessageHandler = nil

// Manager 客户端管理入口
var manager Interface = NewDefaultManager()

func SignIn(oldUid int64, uid int64, device int64) error {
	return manager.ClientSignIn(oldUid, uid, device)
}
func Logout(uid int64, device int64) error {
	return manager.ClientLogout(uid, device)
}
func IsDeviceOnline(uid, device int64) bool {
	return false
}
func IsOnline(uid int64) bool {
	return true
}

// EnqueueMessage Manager.EnqueueMessage 的快捷方法, 预留一个位置对消息入队列进行一些预处理
func EnqueueMessage(uid int64, message *message.Message) error {
	//
	return manager.EnqueueMessage(uid, 0, message)
}

func EnqueueMessageToDevice(uid int64, device int64, message *message.Message) error {
	return manager.EnqueueMessage(uid, device, message)
}

func SetInterfaceImpl(i Interface) {
	manager = i
}

func SetMessageHandler(handler MessageHandler) {
	messageHandleFunc = func(from int64, device int64, message *message.Message) error {
		err := handler(from, device, message)
		if err != nil {
			logger.E("handle message error: %v", err)
		}
		return err
	}
}

var cacheServerInfo *ServerInfo = nil
var cacheInfoExpired = time.Now()

func GetServerInfo(count int) *ServerInfo {
	clientManager, ok := manager.(*DefaultClientManager)
	if ok {
		if cacheInfoExpired.After(time.Now()) {
			return cacheServerInfo
		}
		cacheInfoExpired = time.Now().Add(time.Second * 5)
		info := clientManager.GetManagerInfo()
		cacheServerInfo = &info
		cacheServerInfo.OnlineCli = clientManager.getClient(count)
		return cacheServerInfo
	}
	return &ServerInfo{}
}
