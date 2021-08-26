package client

import (
	"context"
	"go_im/im/client"
	"go_im/im/conn"
	"go_im/im/message"
	"go_im/service/client/pb"
	"go_im/service/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type Client struct {
	rpc pb.ClientServiceClient
	*rpc.BaseClient
}

func NewClient(options *rpc.ClientOptions) *Client {
	ret := &Client{}
	ret.BaseClient = rpc.NewBaseClient(options)
	ret.Init(options)
	client.Manager = ret
	return ret
}

func (c *Client) ClientConnected(conn conn.Connection) int64 {
	return -1
}

func (c *Client) ClientSignIn(oldUid int64, uid int64, device int64) {
	// TODO
}

func (c *Client) UserLogout(uid int64) {
	// TODO
}

func (c *Client) DispatchMessage(from int64, message *message.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.rpc.DispatchMessage(ctx, &pb.UidMessageRequest{
		From:    from,
		Message: wrapMessage(message),
	})

	if err != nil {

	}
	return nil
}

func (c *Client) Api(from int64, message *message.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.rpc.Api(ctx, &pb.UidMessageRequest{
		From:    from,
		Message: wrapMessage(message),
	})

	if err != nil {

	}
}

func (c *Client) EnqueueMessage(uid int64, message *message.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.rpc.EnqueueMessage(ctx, &pb.UidMessageRequest{
		From:    uid,
		Message: wrapMessage(message),
	})

	if err != nil {

	}
}

func (c *Client) IsOnline(uid int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.rpc.IsOnline(ctx, &pb.UidRequest{
		Uid: uid,
	})

	if err != nil {

	}
	return false
}

func (c *Client) AllClient() []int64 {
	// TODO
	return nil
}

func (c *Client) Update() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.rpc.Update(ctx, &emptypb.Empty{})
	if err != nil {

	}
}

func (c *Client) Run() error {
	err := c.Connect()
	c.rpc = pb.NewClientServiceClient(c.Conn)
	return err
}

func wrapMessage(msg *message.Message) *pb.Message {
	return &pb.Message{
		Seq:    msg.Seq,
		Action: string(msg.Action),
		Data:   msg.Data,
	}
}
