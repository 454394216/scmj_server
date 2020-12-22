package sccardtable

import (
	"fmt"
	"math/rand"
	"time"
	"zinx-mj/game/card/boardcard"
	"zinx-mj/game/gamedefine"
	"zinx-mj/game/rule/board"
	"zinx-mj/game/rule/chow"
	"zinx-mj/game/rule/deal"
	"zinx-mj/game/rule/discard"
	"zinx-mj/game/rule/irule"
	"zinx-mj/game/rule/kong"
	"zinx-mj/game/rule/pong"
	"zinx-mj/game/rule/shuffle"
	"zinx-mj/game/rule/ting"
	"zinx-mj/game/rule/win"
	"zinx-mj/game/table/tableoperate"
	"zinx-mj/game/table/tableplayer"
	"zinx-mj/game/table/tablestate"
	"zinx-mj/mjerror"
	"zinx-mj/network/protocol"
	"zinx-mj/player"
	"zinx-mj/util"

	"github.com/aceld/zinx/zlog"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"
)

type ScCardTable struct {
	id           uint32                     // 桌子ID
	players      []*tableplayer.TablePlayer // 房间的玩家
	startTm      int64
	games        int // 游戏局数
	turnSeat     int // 当前回合的玩家
	nextTurnSeat int // 当前操作的玩家

	data         *ScTableData
	event        *ScTableEvent
	board        *boardcard.BoardCard
	stateMachine *tablestate.StateMachine

	boardRule   irule.IBoard
	chowRule    irule.IChow
	discardRule irule.IDiscard
	kongRule    irule.IKong
	pongRule    irule.IPong
	shuffleRule irule.IShuffle
	tingRule    irule.ITing
	winRule     irule.IWin
	dealRule    irule.IDeal
}

func NewTable(tableID uint32, master *tableplayer.TablePlayerData, data *ScTableData) (*ScCardTable, error) {
	t := &ScCardTable{
		id:      tableID,
		startTm: time.Now().Unix(),
		data:    data,
	}
	t.boardRule = board.NewThreeSuitBoard()
	t.chowRule = chow.NewEmptyChow()
	t.discardRule = discard.NewDingQueDiscard()
	t.kongRule = kong.NewGeneralKong()
	t.pongRule = pong.NewGeneralPong()
	t.shuffleRule = shuffle.NewRandomShuffle()
	t.tingRule = ting.NewGeneralRule()
	t.winRule = win.NewGeneralWin()
	t.dealRule = deal.NewGeneralDeal()

	t.initEvent()
	t.initStateMachine()
	_, err := t.Join(master, gamedefine.TABLE_IDENTIY_MASTER|gamedefine.TABLE_IDENTIY_PLAYER)
	if err != nil {
		return t, err
	}
	return t, nil
}

func (s *ScCardTable) initStateMachine() {
	s.stateMachine = tablestate.New(s)
}

func (s *ScCardTable) initEvent() {
	s.event = NewScTableEvent()
	_ = s.event.On(EVENT_JOIN, s.onJoinEvent)
	_ = s.event.On(EVENT_GAME_START, s.onGameStart)
}

func (s *ScCardTable) GetID() uint32 {
	return s.id
}

func (s *ScCardTable) GetPlayerSeat(pid uint64) int {
	for i := range s.players {
		ply := s.players[i]
		if ply.Pid == pid {
			return i
		}
	}
	return len(s.players)
}

func (s *ScCardTable) onGameStart() error {
	// 初始化玩家手牌
	s.initializeHandCard()
	s.games++ // 增加游戏局数

	// 初始化庄家
	s.nextTurnSeat = rand.Intn(s.data.MaxPlayer) // 随机庄家
	// 稍后广播玩家手牌
	msg := &protocol.ScGameTurnStart{}
	// todo 抽象筛子点数rule
	msg.DiePoint = rand.Int31n(6) + 1

	// 广播游戏开始消息
	if err := s.broadCastCommon(protocol.PROTOID_SC_GAME_TURN_START, msg); err != nil {
		zlog.Errorf("broadCast game start failed, err=%s", err)
		return err
	}
	// 广播玩家手牌
	if err := s.broadCastCardInfo(); err != nil {
		zlog.Errorf("broadcast card info failed, err=%s", err)
		return err
	}

	// 初始化座位
	s.UpdateTurnSeat()
	// 初始状态为抽牌状态
	if err := s.stateMachine.SetInitState(tablestate.TABLE_STATE_DRAW); err != nil {
		return err
	}

	return nil
}

func (s *ScCardTable) GetPlayerByPid(pid player.PID) *tableplayer.TablePlayer {
	for _, ply := range s.players {
		if pid == ply.Pid {
			return ply
		}
	}
	return nil
}

func (s *ScCardTable) GetPlayerBySeat(seat int) *tableplayer.TablePlayer {
	if seat >= len(s.players) || seat < 0 {
		return nil
	}
	return s.players[seat]
}

func (s *ScCardTable) Join(plyData *tableplayer.TablePlayerData, identity uint32) (*tableplayer.TablePlayer, error) {
	ply := tableplayer.NewTablePlayer(plyData, s)
	ply.AddIdentity(identity)
	s.players = append(s.players, ply)

	s.event.Add(EVENT_JOIN, ply.Pid)

	return ply, nil
}

func (s *ScCardTable) IsFull() bool {
	return len(s.players) >= s.data.MaxPlayer
}

func (s *ScCardTable) Quit(pid player.PID) error {
	panic("implement me")
}

func (s *ScCardTable) GetStartTime() int64 {
	return s.startTm
}

func (s *ScCardTable) GetTableNumber() uint32 {
	return s.id
}

// 广播同样的消息给所有玩家
func (s *ScCardTable) broadCastCommon(protoID protocol.PROTOID, msg proto.Message) error {
	for _, ply := range s.players {
		if err := util.SendMsg(ply.Pid, protoID, msg); err != nil {
			return fmt.Errorf("braodcast to player failed, pid=%d, protoID=%d", ply.Pid, protoID)
		}
	}
	return nil
}

func (s *ScCardTable) PackPlayerData(ply *tableplayer.TablePlayer) *protocol.TablePlayerData {
	return &protocol.TablePlayerData{
		Pid:         ply.Pid,
		Photo:       0,
		Name:        ply.Name,
		Identity:    ply.Identity,
		OnlineState: uint32(ply.OnlineState),
	}
}

func (s *ScCardTable) PackToPBMsg() proto.Message {
	reply := &protocol.ScScmjTableInfo{}
	reply.TableId = s.id
	reply.StartTime = s.GetStartTime()
	reply.Data = s.data.PackToPBMsg().(*protocol.ScmjData)
	for _, ply := range s.players {
		reply.Players = append(reply.Players, s.PackPlayerData(ply))
	}
	return reply
}

func (s *ScCardTable) onJoinEvent(pid player.PID) error {
	// 通知其它玩家该玩家加入了房间 // 其实应该延迟一帧发送，需要等待其他协议
	msg := &protocol.ScJoinTable{}
	ply := s.GetPlayerByPid(pid)
	if ply == nil {
		return errors.WithStack(mjerror.ErrPlyNotFound)
	}
	msg.Player = s.PackPlayerData(ply)
	msg.SeatIndex = int32(len(s.players)) - 1
	if err := s.broadCastCommon(protocol.PROTOID_SC_JOIN_TABLE, msg); err != nil {
		return err
	}

	// 人满了就开游戏
	if s.IsFull() {
		s.event.Add(EVENT_GAME_START)
	}
	return nil
}

func (s *ScCardTable) initializeHandCard() {
	const MAX_HAND_CARD_NUM = 13
	s.board = s.boardRule.NewBoard()
	s.shuffleRule.Shuffle(s.board.Cards)

	for i := 0; i < s.data.MaxPlayer; i++ {
		s.players[i].InitHandCard(s.board.Cards[:MAX_HAND_CARD_NUM])
		s.board.Cards = s.board.Cards[MAX_HAND_CARD_NUM:]
	}
}

func (s *ScCardTable) broadCastCardInfo() error {
	for _, ply := range s.players {
		cardInfo := s.PackCardInfo(ply.Pid)
		if err := util.SendMsg(ply.Pid, protocol.PROTOID_SC_CARD_INFO, cardInfo); err != nil {
			return err
		}
	}
	return nil
}

func (s *ScCardTable) broadCastDrawCard(pid player.PID, card int) {
	msg := &protocol.ScDrawCard{}
	msg.Pid = pid
	for _, ply := range s.players {
		if ply.Pid == pid { // 摸到的牌只会发给本人
			msg.Card = int32(card)
		} else {
			msg.Card = -1
		}
		if err := util.SendMsg(ply.Pid, protocol.PROTOID_SC_DRAW_CARD, msg); err != nil {
			zlog.Errorf("send msg error:%s", err)
			return
		}
	}
}

func (s *ScCardTable) GetHandCardArray(ply *tableplayer.TablePlayer, pid player.PID) []int {
	// notice: cards必须是手牌的copy, 后面可能会修改
	cards := ply.Hcard.GetCardArray()
	if ply.Pid == pid {
		return cards
	}
	// 不是自己的就仅仅返回一个占位符
	for i := 0; i < len(cards); i++ {
		cards[i] = -1
	}
	return cards
}

func (s *ScCardTable) PackCardInfo(pid player.PID) *protocol.ScCardInfo {
	msg := &protocol.ScCardInfo{}
	msg.TableCard = &protocol.TableCardData{}
	msg.TableCard.Total = int32(len(s.board.CardsTotal))
	msg.TableCard.Left = int32(len(s.board.Cards))

	for _, ply := range s.players {
		cards := s.GetHandCardArray(ply, pid)
		handCards := &protocol.HandCardData{}
		for _, card := range cards {
			handCards.Card = append(handCards.Card, int32(card))
		}
		msg.HandCard = append(msg.HandCard, handCards)
	}

	return msg
}

func (s *ScCardTable) Update(delta time.Duration) {
	s.event.FireAll() // 处理所有的事件
	if err := s.stateMachine.Update(); err != nil {
		zlog.Errorf("update state machine failed, err:%s", err)
	}
}

func (s *ScCardTable) DrawCard() error {
	turnPly := s.GetTurnPlayer()
	card, err := s.board.DrawForward()
	if err != nil {
		return err
	}
	if err = turnPly.DrawCard(card); err != nil {
		return err
	}
	s.broadCastDrawCard(turnPly.Pid, card)
	return nil
}

func (s *ScCardTable) NotifyPlyOperate(ply *tableplayer.TablePlayer) error {
	// todo
	return nil
}

func (s *ScCardTable) GetPlayers() []*tableplayer.TablePlayer {
	return s.players
}

func (s *ScCardTable) GetWinRule() irule.IWin {
	return s.winRule
}

func (s *ScCardTable) GetDiscardRule() irule.IDiscard {
	return s.discardRule
}

func (s *ScCardTable) OnPlyOperate(pid uint64, operate tableoperate.OperateCommand) error {
	ply := s.GetPlayerByPid(pid)
	if ply == nil {
		return errors.Errorf("not found player, pid=%d", pid)
	}
	if !ply.IsOperateValid(operate.OpType) {
		return errors.Errorf("unalid op for ply op=%d, pid=%d", operate.OpType, pid)
	}

	if err := s.stateMachine.GetCurState().OnPlyOperate(pid, operate); err != nil {
		return err
	}

	// 跳过操作不通知玩家
	if err := ply.DoOperate(operate); err != nil {
		return err
	}

	// 更新下一个回合的玩家
	if operate.OpType != tableoperate.OPERATE_PASS {
		turnSeat := s.GetTurnSeat()
		maxPlayer := s.data.MaxPlayer
		plySeat := s.GetPlayerSeat(ply.Pid)
		// 靠后的玩家才会更新
		if util.SeatRelative(plySeat, turnSeat, maxPlayer) > util.SeatRelative(s.nextTurnSeat, turnSeat, maxPlayer) {
			s.SetNextSeat(plySeat)
		}
	}

	// if err := s.broadCastCommon(protocol.PROTOID_CS_DISCARD_CARD); err != nil {
	// 	return err
	// }

	return nil
}

func (s *ScCardTable) GetTurnSeat() int {
	return s.turnSeat
}

func (s *ScCardTable) GetTurnPlayer() *tableplayer.TablePlayer {
	return s.GetPlayerBySeat(s.turnSeat)
}

func (s *ScCardTable) GetNextTurnPlayer() *tableplayer.TablePlayer {
	return s.GetPlayerBySeat(s.nextTurnSeat)
}

// 更新的玩家回合时需要清掉玩家的所有操作
func (s *ScCardTable) UpdateTurnSeat() {
	s.turnSeat = s.nextTurnSeat
	for _, ply := range s.players {
		ply.ClearOperates()
	}
}

func (s *ScCardTable) SetNextSeat(seat int) {
	if seat > s.data.MaxPlayer {
		seat = 0
	}
	s.nextTurnSeat = seat
}

func (s *ScCardTable) AfterDiscard() error {
	turnPly := s.GetTurnPlayer()
	c := turnPly.Hcard.GetLastDiscard()
	for i := range s.players {
		ply := s.players[i]
		if ply.Pid == turnPly.Pid {
			continue
		}
		ops := ply.GetOperateOnOtherTurn(c)
		ply.AddOperate(ops...)
	}
	return nil
}
