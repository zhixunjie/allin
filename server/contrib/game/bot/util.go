package bot

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/allin/server/contrib/game/model"
)

const UserIdPrefix = "bot_"

// GenUserID 根据房间码和序号生成确定性的 bot 用户 ID。
func GenUserID(roomCode string, i int) string {
	return fmt.Sprintf("%v%s_%d", UserIdPrefix, roomCode, i)
}

// GenUserName 返回第 i 个 bot 的显示名。
func GenUserName(i int) string {
	names := []string{
		"Alice", "Bob", "Charlie", "Diana", "Eve",
		"Frank", "Grace", "Henry", "Ivy", "Jack",
	}
	return names[i%len(names)]
}

// IsBotID 判断 userID 是否属于 bot（以 UserIdPrefix 开头）。
func IsBotID(userID string) bool {
	return strings.HasPrefix(userID, UserIdPrefix)
}

// AssignStyle 根据房间风格主题（SetBotType）和 bot 序号，返回单个 bot 的具体风格。
//
//	mixed（默认）: TAG→LAG→Station→Rock 循环
//	aggressive:    TAG→LAG 交替
//	passive:       Rock→Station 交替
//	random:        每个 bot 独立随机
func AssignStyle(roomStyle model.SetBotType, index int) BotStyle {
	switch roomStyle {
	case model.RoomBotTypeAggressive:
		return []BotStyle{BotStyleTag, BotStyleLag}[index%2]
	case model.RoomBotTypePassive:
		return []BotStyle{BotStyleRock, BotStyleStation}[index%2]
	case model.RoomBotTypeRandom:
		return styleOrder[rand.Intn(len(styleOrder))]
	default: // mixed 或空字符串
		return styleOrder[index%len(styleOrder)]
	}
}
