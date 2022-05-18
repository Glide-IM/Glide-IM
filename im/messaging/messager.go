package messaging

import (
	"github.com/panjf2000/ants/v2"
	"go_im/im/api"
	"go_im/im/message"
	"go_im/im/statistics"
	"go_im/pkg/logger"
	"strings"
)

// execPool 100 capacity goroutine pool, 假设每个消息处理需要10ms, 一个协程则每秒能处理100条消息
var execPool *ants.Pool

var messageHandlerFunMap = map[message.Action]func(from int64, device int64, msg *message.Message){
	message.ActionGroupMessageRecall: handleGroupRecallMsg,
	message.ActionChatMessageRecall:  handleChatRecallMessage,
	message.ActionChatMessage:        handleChatMessage,
	message.ActionChatMessageRetry:   handleChatMessage,
	message.ActionChatMessageResend:  handleChatMessage,
	message.ActionGroupMessage:       handleGroupMsg,
	message.ActionCSMessage:          handleCustomerServiceMsg,
	message.ActionAckRequest:         handleAckRequest,
	message.ActionAckGroupMsg:        handleAckGroupMsgRequest,
	message.ActionClientCustom:       handleClientCustom,
}

func init() {
	var err error
	execPool, err = ants.NewPool(50_0000,
		ants.WithNonblocking(true),
		ants.WithPanicHandler(onHandleMessagePanic),
		ants.WithPreAlloc(false),
	)
	if err != nil {
		panic(err)
	}
}

// handleMessage 处理接收到的所有类型消息, 所有消息处理的入口
func handleMessage(from int64, device int64, msg *message.Message) error {
	logger.D("new message: uid=%d, %v", from, msg)
	err := execPool.Submit(func() {
		statistics.SMsgInput()
		h, ok := messageHandlerFunMap[message.Action(msg.GetAction())]
		if ok {
			h(from, device, msg)
			return
		}
		switch msg.GetAction() {
		case message.ActionHeartbeat:
			handleHeartbeat(from, device, msg)
		default:
			if strings.HasPrefix(msg.GetAction(), message.ActionApi) {
				resp, err := api.Handle(from, device, msg)
				if err != nil {
					logger.E("handle api message error %v", err)
				}
				if resp != nil {
					enqueueMessage2Device(from, device, resp)
				}
			} else {
				enqueueMessage(from, message.NewMessage(-1, message.ActionNotifyError, "unknown action"))
				logger.W("receive a unknown action message: " + string(msg.GetAction()))
			}
		}
	})
	if err != nil {
		if err == ants.ErrPoolOverload {
			logger.E("Messaging.MessageHandler goroutine pool is overload")
			return err
		}
		if err == ants.ErrPoolClosed {
			logger.E("Messaging.MessageHandler goroutine pool is closed")
			return err
		}
		enqueueMessage(from, message.NewMessage(-1, message.ActionNotifyError, "internal server error"))
		logger.E("async handle message error %v", err)
	}
	return nil
}

func handleHeartbeat(from int64, device int64, msg *message.Message) {
	// TODO 2021-11-15 处理心跳消息
}

// handleAckRequest 处理接收者收到消息发回来的确认消息
func handleAckRequest(from int64, device int64, msg *message.Message) {
	ackMsg := new(message.AckRequest)
	if !unwrap(from, msg, ackMsg) {
		return
	}
	ackNotify := message.NewMessage(0, message.ActionAckNotify, ackMsg)
	// 通知发送者, 对方已收到消息
	enqueueMessage(ackMsg.From, ackNotify)
}

// handleCustomerServiceMsg 分发客服消息
func handleCustomerServiceMsg(from int64, device int64, msg *message.Message) {
	csMsg := new(message.CustomerServiceMessage)
	if !unwrap(from, msg, csMsg) {
		return
	}
	// 发送消息给客服
	enqueueMessage(csMsg.CsId, msg)
}

func handleClientCustom(from int64, device int64, msg *message.Message) {
	m := new(message.ClientCustom)
	if !unwrap(from, msg, m) {
		return
	}
	m2 := message.NewMessage(0, message.ActionClientCustom, m)
	enqueueMessage(m.To, m2)
}

func onHandleMessagePanic(i interface{}) {
	statistics.SError(i.(error))
	logger.E("handler message panic, %v", i)
}

// unwrap 解包, 反序列化消息包中数据到对象
func unwrap(from int64, msg *message.Message, to interface{}) bool {
	err := msg.DeserializeData(to)
	if err != nil {
		enqueueMessage(from, message.NewMessage(msg.GetSeq(), message.ActionNotifyError, "send message failed"))
		logger.E("sender chat senderMsg %v", err)
		return false
	}
	return true
}
