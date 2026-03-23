package room

import (
	"errors"
	"slices"

	"github.com/allin/server/gmodel"
)

// RoomConfig 是房主创建房间时设定的配置。
type RoomConfig struct {
	SmallBlind    int64             `json:"small_blind"`     // 小盲注
	BigBlind      int64             `json:"big_blind"`       // 大盲注（通常为小盲注的 2 倍）
	MinBuyIn      int64             `json:"min_buy_in"`      // 最小买入额
	MaxBuyIn      int64             `json:"max_buy_in"`      // 最大买入额
	MaxPlayers    int               `json:"max_players"`     // 最大玩家数（2–9）
	ActionTimeSec int               `json:"action_time_sec"` // 每人行动时限（秒），默认 30
	BotCount      int               `json:"bot_count"`       // AI 玩家数量（0 = 无 bot）
	BotType       model.RoomBotType `json:"bot_style"`       // bot 风格主题，默认 mixed
}

// validate 校验房间配置合法性。ActionTimeSec 为 0 时视为使用默认值 30，跳过范围检查。
func (cfg RoomConfig) validate() error {
	if cfg.SmallBlind <= 0 {
		return errors.New("small_blind must be positive")
	}
	if cfg.BigBlind != cfg.SmallBlind*2 {
		return errors.New("big_blind must be 2x small_blind")
	}
	if cfg.MinBuyIn < cfg.BigBlind*10 {
		return errors.New("min_buy_in must be at least 10x big_blind")
	}
	if cfg.MaxBuyIn < cfg.MinBuyIn {
		return errors.New("max_buy_in must be >= min_buy_in")
	}
	if cfg.MaxPlayers < 2 || cfg.MaxPlayers > 9 {
		return errors.New("max_players must be between 2 and 9")
	}
	if cfg.BotCount < 0 || cfg.BotCount >= cfg.MaxPlayers {
		return errors.New("bot_count must be >= 0 and < max_players")
	}
	// 合法值，含空字符串（使用默认 mixed）
	if !slices.Contains(model.AllRoomBotType, cfg.BotType) {
		return errors.New("bot_style must be mixed, aggressive, passive, or random")
	}
	if t := cfg.ActionTimeSec; t != 0 && (t < 5 || t > 120) {
		return errors.New("action_time_sec must be between 5 and 120")
	}
	return nil
}
