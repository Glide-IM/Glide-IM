package gateway

import (
	"github.com/gogo/protobuf/proto"
	"github.com/nsqio/go-nsq"
	"go_im/pkg/mq_nsq"
	"go_im/protobuf/gen/pb_rpc"
)

type msgHandler struct {
}

func (msgHandler) HandleMessage(msg *nsq.Message) error {
	if len(msg.Body) == 0 {
		return nil
	}
	m := pb_rpc.NSQUserMessage{}
	err := proto.Unmarshal(msg.Body, &m)
	if err != nil {
		return err
	}
	msg.Finish()
	//client.EnqueueMessage(m.Uid, &message.Message{CommMessage: m.Message})
	return nil
}

func RegisterGateway(s *Server, config *mq_nsq.NSQConsumerConfig) error {
	var err error
	config.Channel = "ch"
	config.Topic = topicPrefix + s.Options.Addr
	consumer, err = mq_nsq.NewConsumer(config)
	if err != nil {
		return err
	}
	consumer.AddHandler(&msgHandler{})
	return consumer.Connect()
}
