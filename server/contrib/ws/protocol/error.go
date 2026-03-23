package protocol

// SkErrCode 是服务端错误码枚举。
type SkErrCode string

const (
	SkErrRoomFull          SkErrCode = "room_full"          // 房间已满，无法加入
	SkErrInvalidBuyIn      SkErrCode = "invalid_buy_in"     // 买入金额不在允许范围内
	SkErrUserNotFound      SkErrCode = "user_not_found"     // 找不到对应用户记录
	SkErrInsufficientChips SkErrCode = "insufficient_chips" // 账户余额不足以完成操作
	SkErrServerError       SkErrCode = "server_error"       // 服务端内部错误
	SkErrBadPayload        SkErrCode = "bad_payload"        // 消息体格式错误或无法解析
	SkErrInvalidAction     SkErrCode = "invalid_action"     // 行动不合规则（如不是你的回合）
	SkErrHandInProgress    SkErrCode = "hand_in_progress"   // 手牌进行中，该操作只能在手牌间隙执行
	SkErrNotSeated         SkErrCode = "not_seated"         // 玩家未在座位上
	SkErrInvalidAmount     SkErrCode = "invalid_amount"     // 金额参数非法（如小于等于零）
)

var errMessages = map[SkErrCode]string{
	SkErrRoomFull:          "room is full",
	SkErrInvalidBuyIn:      "invalid buy-in amount",
	SkErrUserNotFound:      "user not found",
	SkErrInsufficientChips: "insufficient chips",
	SkErrServerError:       "internal server error",
	SkErrBadPayload:        "invalid payload",
	SkErrInvalidAction:     "invalid action",
	SkErrHandInProgress:    "hand is in progress",
	SkErrNotSeated:         "you are not seated",
	SkErrInvalidAmount:     "amount must be positive",
}

// Message 返回该错误码对应的默认描述。
func (c SkErrCode) Message() string {
	if msg, ok := errMessages[c]; ok {
		return msg
	}
	return string(c)
}

// ErrorPayload 承载对客户端命令的错误响应。
type ErrorPayload struct {
	Code    SkErrCode `json:"code"`    // 错误码，见 SkErrCode 枚举
	Message string    `json:"message"` // 可读的错误描述
	RefSeq  int64     `json:"ref_seq"` // 触发该错误的客户端命令序列号（用于关联请求）
}
