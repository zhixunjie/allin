package protocol

// ErrCode 是服务端错误码枚举。
type ErrCode string

const (
	ErrRoomFull          ErrCode = "room_full"          // 房间已满，无法加入
	ErrInvalidBuyIn      ErrCode = "invalid_buy_in"     // 买入金额不在允许范围内
	ErrUserNotFound      ErrCode = "user_not_found"     // 找不到对应用户记录
	ErrInsufficientChips ErrCode = "insufficient_chips" // 账户余额不足以完成操作
	ErrServerError       ErrCode = "server_error"       // 服务端内部错误
	ErrBadPayload        ErrCode = "bad_payload"        // 消息体格式错误或无法解析
	ErrInvalidAction     ErrCode = "invalid_action"     // 行动不合规则（如不是你的回合）
	ErrHandInProgress    ErrCode = "hand_in_progress"   // 手牌进行中，该操作只能在手牌间隙执行
	ErrNotSeated         ErrCode = "not_seated"         // 玩家未在座位上
	ErrInvalidAmount     ErrCode = "invalid_amount"     // 金额参数非法（如小于等于零）
)

var errMessages = map[ErrCode]string{
	ErrRoomFull:          "room is full",
	ErrInvalidBuyIn:      "invalid buy-in amount",
	ErrUserNotFound:      "user not found",
	ErrInsufficientChips: "insufficient chips",
	ErrServerError:       "internal server error",
	ErrBadPayload:        "invalid payload",
	ErrInvalidAction:     "invalid action",
	ErrHandInProgress:    "hand is in progress",
	ErrNotSeated:         "you are not seated",
	ErrInvalidAmount:     "amount must be positive",
}

// Message 返回该错误码对应的默认描述。
func (c ErrCode) Message() string {
	if msg, ok := errMessages[c]; ok {
		return msg
	}
	return string(c)
}

// ErrorPayload 承载对客户端命令的错误响应。
type ErrorPayload struct {
	Code    ErrCode `json:"code"`    // 错误码，见 ErrCode 枚举
	Message string  `json:"message"` // 可读的错误描述
	RefSeq  int64   `json:"ref_seq"` // 触发该错误的客户端命令序列号（用于关联请求）
}
