package protocol

import "encoding/json"

// InboundMessage 将客户端命令与发送者元数据包装在一起，投递到 RoomConn.Inbound channel。
type InboundMessage struct {
	SenderID    string      // 发送者的用户 ID（字符串形式）
	DisplayName string      // 发送者的显示名称，用于聊天/日志
	Env         CmdEnvelope // 原始命令信封，含命令类型和载荷
}

// CmdEnvelope 是客户端 → 服务端的命令包装。
type CmdEnvelope struct {
	Type    CmdType         `json:"type"`    // 命令类型，见 CmdType 枚举
	Seq     int64           `json:"seq"`     // 客户端递增序列号，用于关联响应
	Ts      int64           `json:"ts"`      // 客户端发送时间戳（Unix 毫秒）
	Payload json.RawMessage `json:"payload"` // 具体命令载荷，按 Type 解析为对应 Cmd 结构体
}

// CmdType 是客户端 → 服务端的命令类型。
type CmdType string

const (
	CmdJoinRoom   CmdType = "join_room"   // JoinRoomCmd 加入房间（携带买入额）；
	CmdAction     CmdType = "action"      // ActionCmd 玩家行动（fold/check/call/bet/raise/all_in）；
	CmdChat       CmdType = "chat"        // ChatCmd 发送聊天消息；
	CmdSitOut     CmdType = "sit_out"     // SitOutCmd 离座/归座切换；
	CmdLeaveTable CmdType = "leave_table" // 无载荷 主动离桌（仅限手牌间隙）；
	CmdDisconnect CmdType = "disconnect"  // 无载荷 客户端主动断开（优雅离线）；
	CmdReady      CmdType = "ready"       // 无载荷 结算画面点击"准备下一局"；
)

// 定义所有客户端 → 服务端的命令载荷结构体。

// JoinRoomCmd 是 CmdJoinRoom 的载荷。
type JoinRoomCmd struct {
	RoomCode  string `json:"room_code"`  // 要加入的房间码
	SeatIndex *int   `json:"seat_index"` // 可选的首选座位号，nil 表示自动分配
	BuyIn     int64  `json:"buy_in"`     // 买入金额；0 表示使用 MaxBuyIn
}

// ActionCmd 是 CmdAction 的载荷。
type ActionCmd struct {
	Action string `json:"action"` // 行动类型：fold/check/call/bet/raise/all_in
	Amount int64  `json:"amount"` // 行动金额（bet/raise 时有效，其余为 0）
}

// ChatCmd 是 CmdChat 的载荷。
type ChatCmd struct {
	Text string `json:"text"` // 聊天消息正文
}

// SitOutCmd 是 CmdSitOut 的载荷。
type SitOutCmd struct {
	SitOut bool `json:"sit_out"` // true = 请求离座，false = 请求归座
}
