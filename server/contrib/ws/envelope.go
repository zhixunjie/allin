package ws

import (
	"encoding/json"
	"time"
)

// Envelope 是所有服务端 → 客户端消息的通用包装。
type Envelope struct {
	Type    MsgType         `json:"type"`    // 事件类型，见 MsgType 枚举
	Seq     int64           `json:"seq"`     // 服务端递增序列号，客户端可用于检测消息丢失
	Ts      int64           `json:"ts"`      // 服务端发送时间戳（Unix 毫秒）
	Payload json.RawMessage `json:"payload"` // 具体事件载荷，按 Type 解析为对应 Payload 结构体
}

// NewEnvelope 构建一个带当前时间戳的服务端事件信封。
func NewEnvelope(msgType MsgType, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Type:    msgType,
		Ts:      time.Now().UnixMilli(),
		Payload: raw,
	}, nil
}

// MustNewEnvelope 类似 NewEnvelope，但在序列化错误时 panic（对已知类型安全）。
func MustNewEnvelope(msgType MsgType, payload any) Envelope {
	e, err := NewEnvelope(msgType, payload)
	if err != nil {
		panic(err)
	}
	return e
}
