package bot

import (
	"fmt"
	"strings"
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
