package state

import (
	"github.com/allin/server/contrib/game/model"
	"github.com/allin/server/contrib/room"
)

// GameSnapshot 是发送给客户端的完整状态，在重连或首次入座时下发。
type GameSnapshot struct {
	Street     string          `json:"street"`      // 当前街道名称（idle/preflop/…）
	HandNum    int             `json:"hand_num"`    // 当前手牌编号（从 1 递增）
	Community  []string        `json:"community"`   // 公共牌字符串列表
	Seats      []SeatSnapshot  `json:"seats"`       // 所有已入座玩家的状态列表
	Pot        int64           `json:"pot"`         // 当前总底池
	DealerSeat int             `json:"dealer_seat"` // 庄家座位号
	ActionSeat int             `json:"action_seat"` // 当前待行动座位号；-1 表示无
	CurrentBet int64           `json:"current_bet"` // 本回合最高下注额
	MinRaise   int64           `json:"min_raise"`   // 最低加注增量
	Config     room.RoomConfig `json:"config"`      // 房间配置（盲注/买入等）
}

// SeatSnapshot 是 GameSnapshot 中单个座位的状态。
type SeatSnapshot struct {
	SeatIndex    int      `json:"seat_index"`             // 座位号 0–8
	UserID       string   `json:"user_id"`                // 玩家唯一 ID
	DisplayName  string   `json:"display_name"`           // 显示名称
	Stack        int64    `json:"stack"`                  // 桌面筹码余额
	Bet          int64    `json:"bet"`                    // 本街已下注金额
	Folded       bool     `json:"folded"`                 // 是否已弃牌
	AllIn        bool     `json:"all_in"`                 // 是否全押
	SitOut       bool     `json:"sit_out"`                // 是否离座
	Disconnected bool     `json:"disconnected,omitempty"` // 是否断线
	IsBot        bool     `json:"is_bot,omitempty"`       // 是否为 AI bot
	Hole         []string `json:"hole,omitempty"`         // 底牌（仅对请求玩家自身可见）
}

// Snapshot 构建 GameSnapshot，仅为 viewerID 填充手牌。
func (gs *GameStateMachine) Snapshot(viewerID string) GameSnapshot {
	snap := GameSnapshot{
		Street:     gs.Street.String(),
		HandNum:    gs.HandNum,
		Community:  CardsToStrings(gs.Community),
		Pot:        gs.TotalPot(),
		DealerSeat: gs.DealerSeat,
		ActionSeat: gs.ActionSeat,
		CurrentBet: gs.CurrentBet,
		MinRaise:   gs.MinRaise,
		Config:     gs.Config,
	}
	for _, p := range gs.Seats {
		if p == nil {
			continue
		}
		ss := SeatSnapshot{
			SeatIndex:    p.SeatIndex,
			UserID:       p.UserID,
			DisplayName:  p.DisplayName,
			Stack:        p.Stack,
			Bet:          p.Bet,
			Folded:       p.Folded,
			AllIn:        p.AllIn,
			SitOut:       p.SitOut,
			Disconnected: p.Disconnected,
			IsBot:        p.IsBot,
		}
		if p.UserID == viewerID && gs.Street != model.StreetIdle {
			ss.Hole = []string{p.Hole[0].String(), p.Hole[1].String()}
		}
		snap.Seats = append(snap.Seats, ss)
	}
	return snap
}
