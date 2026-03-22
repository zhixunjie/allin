# contrib/ws — WebSocket 连接管理与协议定义

## 目录结构

```
contrib/ws/
├── README.md       本文件，包说明与目录结构
├── room_conn.go    RoomConn：管理单个房间内所有客户端连接，提供 Broadcast / SendTo
├── client.go       Client：单条 WebSocket 长连接，负责读写泵（ReadPump / WritePump）
├── handler.go      Handler：HTTP 升级入口，维护 RoomConn 生命周期，注入引擎启动回调
├── envelope.go     消息信封：MsgType / CmdType 枚举，Envelope / CmdEnvelope 包装，NewEvent / MustEvent 工厂函数
├── payload.go      服务端 → 客户端事件载荷结构体（ConnectedPayload、ActionRequiredPayload 等）
├── command.go      客户端 → 服务端命令载荷结构体（JoinRoomCmd、ActionCmd 等）
└── error.go        错误码枚举（ErrCode）及错误载荷（ErrorPayload）
```

## 数据流

```
客户端 WebSocket
  │  (JSON CmdEnvelope)
  ▼
Client.ReadPump
  │  (InboundMessage)
  ▼
RoomConn.Inbound  ◄──────────────────────────────────────────┐
  │                                                          │
  ▼                                                          │
Engine.Run()  ──► 处理命令 ──► RoomConn.Broadcast / SendTo  │
                                       │                     │
                              (JSON Envelope)                │
                                       │                     │
                                       ▼                     │
                              Client.WritePump ──► 客户端 WebSocket
```

## 关键设计

- 每个房间对应一个 RoomConn，Handler 负责按房间码创建和销毁。
- Engine 是单 goroutine，通过 RoomConn.Inbound channel 接收命令，避免锁竞争。
- GameSnapshot 在 ConnectedPayload 中以 `any` 类型传递，防止 `ws ↔ game` 循环导入。
