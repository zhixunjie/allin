# AllIn — 德州扑克在线对战

全栈 No-Limit Hold'em 游戏，支持多人实时对战与 AI 陪玩。

![演示](docs/images/output.webp)

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 · Hertz · sqlx + MySQL · JWT |
| 前端 | Vite + React 18 · PixiJS v8 · Zustand |
| 通信 | WebSocket（gorilla/websocket） |
| 部署 | Docker + Docker Compose + nginx |

## 快速启动

**Docker（推荐）：**

```bash
make up   # 访问 http://localhost
```

**本地开发：**

```bash
# 后端（需要 MySQL 127.0.0.1:13306，root 无密码）
cd server && go run ./base/

# 前端
cd ui-web && npm install && npm run dev
```

## 测试账号

密码均为 `123456`，初始筹码各 10,000（`test1` ~ `test4`）。

## 文档

- [项目详情](docs/PROJECT.md)
- [设计规范](docs/DESIGN.md)
- [功能进度](docs/PROGRESS.md)