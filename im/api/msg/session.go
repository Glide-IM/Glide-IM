package msg

import (
	"go_im/im/api/comm"
	"go_im/im/api/router"
	"go_im/im/dao/msgdao"
	"go_im/im/message"
	"time"
)

func (*MsgApi) ReadMessage(ctx *route.Context, request *ReadMessageRequest) error {
	//err := msgdao.SessionDaoImpl.CleanUserSessionUnread(ctx.Uid, request.To, request.To)
	//if err != nil {
	//	return comm.NewDbErr(err)
	//}
	return nil
}

func (*MsgApi) GetOrCreateSession(ctx *route.Context, request *SessionRequest) error {
	session, err := msgdao.SessionDaoImpl.GetSession(ctx.Uid, request.To)
	if err != nil {
		return comm.NewDbErr(err)
	}
	if session.SessionId == "" {
		se, err := msgdao.SessionDaoImpl.CreateSession(ctx.Uid, request.To, time.Now().Unix())
		if err != nil {
			return comm.NewDbErr(err)
		}
		session = se
	}

	ctx.Response(message.NewMessage(ctx.Seq, comm.ActionSuccess, SessionResponse{
		Uid1:     session.Uid,
		Uid2:     session.Uid2,
		LastMid:  session.LastMID,
		UpdateAt: session.UpdateAt,
		CreateAt: session.CreateAt,
	}))

	return nil
}

func (a *MsgApi) GetRecentSessions(ctx *route.Context) error {
	now := time.Now().Unix() + 100
	session, err := msgdao.SessionDaoImpl.GetRecentSession(ctx.Uid, now, 100)
	if err != nil {
		return comm.NewDbErr(err)
	}
	//goland:noinspection GoPreferNilSlice
	sr := []*SessionResponse{}
	for _, s := range session {
		to := s.Uid2
		if s.Uid2 == ctx.Uid {
			to = s.Uid
		}

		sr = append(sr, &SessionResponse{
			Uid2:     s.Uid,
			Uid1:     s.Uid2,
			To:       to,
			LastMid:  s.LastMID,
			UpdateAt: s.UpdateAt,
			CreateAt: s.CreateAt,
		})
	}
	ctx.Response(message.NewMessage(ctx.Seq, comm.ActionSuccess, sr))
	return nil
}
