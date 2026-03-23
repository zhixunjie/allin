# contrib/ws — WebSocket 连接管理与协议定义

## 目录结构

```
contrib/ws/
├── README.md           本文件，包说明与目录结构
├── room_conn.go        RoomConn：管理单个房间内所有客户端连接，提供 Broadcast / SendTo
├── client.go           Client：单条 WebSocket 长连接，负责读写泵（ReadPump / WritePump）
├── handler.go          Handler：HTTP 升级入口，维护 RoomConn 生命周期，注入引擎启动回调
├── error.go            错误码枚举（ErrCode）及错误载荷（ErrorPayload）
└── protocol/
    ├── command.go      客户端 → 服务端：CmdEnvelope / CmdType 枚举、InboundMessage，及所有 Cmd 结构体
    └── payload.go      服务端 → 客户端：Envelope / MsgType 枚举、NewEnvelope / MustNewEnvelope 工厂，及所有 Payload 结构体
```

## 数据流

```
客户端 WebSocket
  │  (JSON CmdEnvelope)
  ▼
Client.ReadPump
  │  (protocol.InboundMessage)
  ▼
RoomConn.Inbound  ◄──────────────────────────────────────────┐
  │                                                          │
  ▼                                                          │
Engine.Run()  ──► 处理命令 ──► RoomConn.Broadcast / SendTo  │
                                       │                     │
                              (JSON protocol.Envelope)       │
                                       │                     │
                                       ▼                     │
                              Client.WritePump ──► 客户端 WebSocket
```

## 关键设计

- 每个房间对应一个 RoomConn，Handler 负责按房间码创建和销毁。
- Engine 是单 goroutine，通过 RoomConn.Inbound channel 接收命令，避免锁竞争。
- 协议类型统一放在 `protocol/` 子包，`ws` 包本身不定义任何消息结构。
- GameSnapshot 在 ConnectedPayload 中以 `any` 类型传递，防止 `ws ↔ game` 循环导入。
