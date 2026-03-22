package engine

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	bizdao "github.com/allin/server/base/biz/dao"
	bizmodel "github.com/allin/server/base/biz/model"
	"github.com/allin/server/contrib/game/state"
	"github.com/allin/server/contrib/ws"
	"github.com/allin/server/contrib/ws/protocol"
)

// saveHandHistory 异步将手牌结果写入 DB，不阻塞引擎 goroutine。
func (e *Engine) saveHandHistory(resultJSON json.RawMessage) {
	type playerSnap struct {
		PlayerID    string `json:"player_id"`
		DisplayName string `json:"display_name"`
		SeatIndex   int    `json:"seat_index"`
		Stack       int64  `json:"stack"`
	}
	var players []playerSnap
	for _, p := range e.gs.Seats {
		if p != nil {
			players = append(players, playerSnap{
				PlayerID:    p.UserID,
				DisplayName: p.DisplayName,
				SeatIndex:   p.SeatIndex,
				Stack:       p.Stack,
			})
		}
	}
	playersJSON, _ := json.Marshal(players)
	actionsJSON, _ := json.Marshal(e.handActions)
	roomID := e.room.ID
	handNum := e.gs.HandNum

	go func() {
		err := bizdao.HandHistoryDao.Save(bizmodel.HandHistoryRecord{
			RoomID:      roomID,
			HandNum:     handNum,
			PlayersJSON: playersJSON,
			ActionsJSON: actionsJSON,
			ResultJSON:  resultJSON,
			PlayedAt:    time.Now(),
		})
		if err != nil {
			slog.Error("game: failed to save hand history", "room", e.room.Code, "hand", handNum, "err", err)
		}
	}()
}

// nextEligibleSeatAfter 在 eligible 列表中找到座位号严格大于 from 的下一个座位（循环）。
// from == -1 时返回第一个 eligible 玩家的座位；用于庄家/盲注/行动顺序推进。
func (e *Engine) nextEligibleSeatAfter(from int, eligible []*state.Player) int {
	if from == -1 {
		if len(eligible) > 0 {
			return eligible[0].SeatIndex
		}
		return 0
	}
	for i := 1; i <= 9; i++ {
		idx := (from + i) % 9
		for _, p := range eligible {
			if p.SeatIndex == idx {
				return idx
			}
		}
	}
	return eligible[0].SeatIndex
}

// sendSnapshot 向指定玩家发送 connected 事件，包含当前完整游戏快照和账户余额。
func (e *Engine) sendSnapshot(userID string) {
	snap := e.gs.Snapshot(userID)
	payload := ws.ConnectedPayload{
		PlayerID:     userID,
		DisplayName:  e.rc.DisplayName(userID),
		RoomCode:     e.room.Code,
		GameSnapshot: &snap,
	}
	if uid, err := strconv.ParseInt(userID, 10, 64); err == nil {
		if u, err := bizdao.UserDao.GetByID(uid); err == nil {
			payload.ChipBalance = u.ChipBalance
		}
	}
	e.rc.SendTo(userID, ws.MustEvent(ws.TypeConnected, payload))
}

// sendError 向指定玩家发送错误事件。
// msgOverride 可选，不传则使用 ErrCode 的默认描述。
func (e *Engine) sendError(userID string, code ws.ErrCode, refSeq int64, msgOverride ...string) {
	msg := code.Message()
	if len(msgOverride) > 0 && msgOverride[0] != "" {
		msg = msgOverride[0]
	}
	e.rc.SendTo(userID, ws.MustEvent(ws.TypeError, ws.ErrorPayload{
		Code: code, Message: msg, RefSeq: refSeq,
	}))
}

// max64 返回两个 int64 中的较大值。
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
