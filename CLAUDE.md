# AllIn — Claude 项目说明

## 项目结构

```
allin/
├── server/             # Go 后端（单 Go module）
│   ├── go.mod          # module github.com/allin/server
│   ├── base/           # 微服务主体
│   │   ├── main.go
│   │   ├── router.go
│   │   ├── config.yaml
│   │   └── biz/
│   │       ├── dao/      # DB 访问层（sqlx）
│   │       ├── handler/  # HTTP 处理器（Hertz）
│   │       ├── model/    # 数据模型
│   │       ├── mw/       # 中间件（JWT）
│   │       └── service/  # 业务逻辑
│   └── contrib/        # 共享基础包
│       ├── auth/         # JWT + bcrypt
│       ├── eval/         # 手牌评估器
│       ├── game/         # 游戏引擎（状态机）
│       ├── room/         # 房间模型 + 内存管理
│       └── ws/           # WebSocket Hub + Client
├── ui-web/             # 前端（Vite + React + PixiJS v8）
└── docs/               # 设计文档和进度追踪
```

## 本地开发启动

### 前置条件
- MySQL 8.0 运行在 `127.0.0.1:13306`，root 无密码
- 连接命令：`mysql -h 127.0.0.1 -P 13306 -u root`
- 数据库 `allin` 已存在，表结构通过 `docs/sql/allin.sql` 手动维护（无 AutoMigrate）

### 后端（server/）
```bash
cd server
go run ./base/
```
- 配置文件：`server/base/config.yaml`（Viper 加载）
- 监听 `:8080`
- 健康检查：`curl http://localhost:8080/health`

### 前端（ui-web/）
```bash
cd ui-web
npm run dev
```
- 默认 `:5173`，端口占用时顺延（5174、5175…）
- `/api` 代理到 `http://localhost:8080`（含 WebSocket）

### 验证后端是否已在运行
```bash
curl http://localhost:8080/health
```

## Docker 一键部署

### 前置条件
- Docker + Docker Compose（无需本地 Go / Node / MySQL）

### 启动
```bash
make up          # 等价于 docker compose up -d
```

首次运行会自动构建镜像（Go 编译 + npm build），完成后访问 `http://localhost`。

### 常用命令

| 命令 | 说明 |
|------|------|
| `make up` | 启动所有服务 |
| `make down` | 停止容器（保留数据） |
| `make down-v` | 停止容器并清空数据库 |
| `make build` | 代码更新后重新构建镜像 |
| `make logs` | 查看实时日志 |
| `make ps` | 查看服务状态 |

### 服务构成

| 服务 | 镜像 | 说明 |
|------|------|------|
| `mysql` | `mysql:8.0` | 自动初始化 schema（`docs/sql/allin.sql`）+ 测试账号（`docs/sql/seed.sql`） |
| `server` | 本地构建（distroless） | Go 后端，监听 `:8080`，使用 `server/base/config.docker.yaml` |
| `web` | 本地构建（nginx:alpine） | 前端静态文件 + 反代 `/api` 到 `server:8080` |

### 配置说明
- **本地开发**：使用 `server/base/config.yaml`（指向 `127.0.0.1:13306`）
- **Docker 部署**：使用 `server/base/config.docker.yaml`（指向 `mysql:3306`，已打包进镜像）
- JWT Secret 默认为 `change-me-in-production`，生产环境请修改 `config.docker.yaml`

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.25 |
| HTTP 框架 | CloudWeGo Hertz v0.10.4 |
| WebSocket | gorilla/websocket（通过 hertz common/adaptor 桥接） |
| 数据库驱动 | jmoiron/sqlx + go-sql-driver/mysql |
| 配置 | spf13/viper（config.yaml） |
| 鉴权 | JWT HS256，7 天有效期 |
| 前端构建 | Vite 5 + TypeScript |
| UI | React 18 + CSS Modules |
| 2D 渲染 | PixiJS v8（WebGL，程序化渲染，无外部图片） |
| 状态管理 | Zustand |
| 路由 | React Router v6 |

## 关键设计决策

- **分层架构**：`handler → service → dao`，与 shopman/server/base 完全对齐。
- **无循环依赖**：`contrib/game` 导入 `contrib/ws/room`；`contrib/ws` 不导入 `contrib/game`。通过 `ws.Handler.SetEngineStarter(func)` 回调在 `main.go` 中注入 Engine 工厂。
- **游戏引擎**：单 goroutine，`select` 监听 `hub.Inbound` + 可重置 timer channel（`nil` channel 不触发 select）。
- **座位旋转**：PixiJS TableScene 将本地玩家始终旋转到底部（display index 0）。
- **手牌评估**：纯 Go 枚举 C(7,5)=21 种五牌组合，`HandRanks.dat` 为可选加速路径。
- **边底池**：按 TotalBet 升序迭代，逐层计算各玩家贡献。
- **买入生命周期**：玩家加入房间时从 `chip_balance` 扣除买入额（`biz/dao.UserDao.AdjustChips`）；断线时将剩余 stack 归还账户。

## 测试账号

密码均为 `123456`，初始筹码各 10,000。

| 账号 | 显示名 |
|------|--------|
| test1 | 测试玩家1 |
| test2 | 测试玩家2 |
| test3 | 测试玩家3 |
| test4 | 测试玩家4 |

多窗口测试：用多个浏览器窗口或无痕模式分别登录不同账号，一人创建房间，其余用房间码加入。

## 进度
详见 `docs/PROGRESS.md`。当前阶段：Phase 17（准备系统）已完成。

## Docs Conventions

- 文档存放目录：`docs`
- **SQL 文件**：所有 DDL 统一存放在 `docs/sql/`
- **SQL 备份文件**：如果我要求从本地数据使用 mysqldump 导出数据库备份文件，路径使用：`docs/sql/bak`，导出命令为：

  ```
  mysqldump --databases <数据库> --set-gtid-purged=OFF \
  --host=<主机> \
  --port=<端口> -u<用户名> -p<密码>
  ```

## Infrastructure

### MySQL

- Host: `127.0.0.1:13306` / User: `root` / Password: (empty)
- 数据库：`allin`

### Redis

- Addr: `127.0.0.1:7000` / Password: (empty)（Cluster 模式）

### Kafka

- Brokers: `127.0.0.1:9092`

---

## Data Model Conventions

- **Struct Tags**：同时使用 `json` 和 `db` tag（sqlx 扫描需要 `db`）
- **ID 类型**：用户/房间 ID 均为 `string`（UUID v4）
- **错误处理**：`fmt.Errorf("xxx failed: %w", err)`
- **错误不忽略**：禁止用 `_` 丢弃 `error`；后台 goroutine 中用 `slog.Error` 记录

### Go Struct 字段注释规范

服务端 Go struct 的每个字段（含私有字段）**必须**在字段后面写行内注释，说明字段含义、取值范围或使用场景。

```go
// ✅ 正确
type Player struct {
    ID           string  // 玩家唯一 ID（UUID v4）
    DisplayName  string  // 显示名
    Stack        int64   // 桌面筹码余额（单位：分）
    Bet          int64   // 本街已下注金额
    Folded       bool    // 是否已弃牌
    Disconnected bool    // 是否处于断线保留座位状态
}

// ❌ 错误（缺少注释）
type Player struct {
    ID          string
    DisplayName string
    Stack       int64
}
```

### 前端数据实体字段注释规范

前端 TypeScript 中声明的数据实体（`interface` / `type`）的每个字段，**必须**在字段后面写行内注释，说明字段含义、取值范围或使用场景。

```ts
// ✅ 正确
interface SeatSnapshot {
  seat_index: number      // 座位号 0–8
  user_id: string         // 玩家唯一 ID（bot 以 "bot_" 开头）
  stack: number           // 桌面筹码余额（未参与下注部分）
  bet: number             // 本街已下注金额
  folded: boolean         // 是否已弃牌
}

// ❌ 错误（缺少注释）
interface SeatSnapshot {
  seat_index: number
  user_id: string
  stack: number
}
```

---

## Code Patterns

### 分层模板

```go
// biz/dao/xxx_dao.go
type xxxDao struct{}
func (d *xxxDao) GetByID(id string) (*model.XXX, error) {
    item := &model.XXX{}
    err := DBM.Get(item, `SELECT * FROM xxx WHERE id = ?`, id)
    return item, err
}

// biz/service/xxx_svc.go
type XxxSvc struct{}
func NewXxxSvc() *XxxSvc { return &XxxSvc{} }
func (svc *XxxSvc) GetByID(id string) (*model.XXX, error) {
    return dao.XxxDao.GetByID(id)
}

// biz/handler/xxx.go
var Xxx XxxHandler
type XxxHandler struct{}
func (*XxxHandler) Get(ctx context.Context, c *app.RequestContext) {
    id := c.Param("id")
    item, err := service.Xxx.GetByID(id)
    if err != nil {
        c.JSON(404, map[string]string{"error": "not found"})
        return
    }
    c.JSON(200, item)
}
```

### 路由注册（router.go）

```go
func register(h *server.Hertz, ...) {
    api := h.Group("/api")
    api.GET("/xxx/:id", handler.Xxx.Get)
    api.POST("/xxx", mw.JWTMiddleware(), handler.Xxx.Create)
}
```

## Database Operations

> 所有 DB 访问通过 DAO；INSERT/UPDATE 优先使用 NamedExec 风格

### MySQL JSON Scan/Value

```go
type Extend struct {
    Uid int64 `json:"uid,omitempty"`
}

func (m Extend) Value() (driver.Value, error) { return json.Marshal(m) }

func (m *Extend) Scan(val interface{}) error {
    switch v := val.(type) {
    case []byte:
        if len(v) == 0 { return nil }
        return json.Unmarshal(v, &m)
    case string:
        if len(v) == 0 { return nil }
        return json.Unmarshal([]byte(v), &m)
    default:
        return errors.New("unsupported type")
    }
}
```

### Query Row / Rows

```go
// 单行
func (d *XXXXDao) GetXXXX(ctx context.Context, id int64) (item *model.XXXX, err error) {
    item = new(model.XXXX)
    query := fmt.Sprintf(`SELECT * FROM %v WHERE id = ?`, model.TableNameXXXX)
    err = SLAVE.Unsafe().GetContext(ctx, item, query, id)
    return
}

// IN 多行
func (d *XXXXDao) GetByIds(ctx context.Context, ids []int64) (list []*model.XXXX, err error) {
    if len(ids) == 0 { return }
    query := fmt.Sprintf(`SELECT * FROM %s WHERE id IN (?)`, model.TableNameXXXX)
    query, args, err := sqlx.In(query, ids)
    if err != nil { return }
    query = SLAVE.Rebind(query)
    err = SLAVE.Unsafe().SelectContext(ctx, &list, query, args...)
    return
}
```

### Insert (NamedExec)

```go
func (d *XXXXDao) Create(ctx context.Context, item *model.XXXX) (err error) {
    insertFields := `name, created_at, updated_at`
    valuesFields := `:name, :created_at, :updated_at`
    query := fmt.Sprintf(`INSERT INTO %v (%v) VALUES (%v)`, model.TableNameXXXX, insertFields, valuesFields)
    _, err = MASTER.NamedExecContext(ctx, query, item)
    return
}
```

### Update (NamedExec)

```go
func (d *XXXXDao) Update(ctx context.Context, item *model.XXXX) (rowAffected int64, err error) {
    updateFields := `name = :name, updated_at = :updated_at`
    query := fmt.Sprintf(`UPDATE %v SET %v WHERE id = :id`, model.TableNameXXXX, updateFields)
    result, err := MASTER.NamedExecContext(ctx, query, item)
    if err != nil { return }
    rowAffected, _ = result.RowsAffected()
    return
}

// 带状态检查的状态变更
func (d *XXXXDao) UpdateStatus(ctx context.Context, id int64, srcStatus, dstStatus int, ts int64) (err error) {
    query := fmt.Sprintf(`UPDATE %v SET status=?, updated_at=? WHERE id=? AND status=?`, model.TableNameXXXX)
    result, err := MASTER.ExecContext(ctx, query, dstStatus, ts, id, srcStatus)
    if err != nil { return }
    if rowAffected, _ := result.RowsAffected(); rowAffected == 0 {
        err = fmt.Errorf("status not change: %v", id)
    }
    return
}
```

### Soft Delete

```go
func (d *XXXXDao) Delete(ctx context.Context, id int64) (err error) {
    ts := gutil.GetNow()
    query := fmt.Sprintf(`UPDATE %v SET updated_at=?, deleted_at=? WHERE id=?`, model.TableNameXXXX)
    _, err = MASTER.ExecContext(ctx, query, ts, ts, id)
    return
}
```

### Batch Update (JOIN)

```go
var selects []string
var args []any
for _, p := range params {
    selects = append(selects, "SELECT ? AS id, ? AS delta")
    args = append(args, p.ID, p.Delta)
}
subQuery := strings.Join(selects, " UNION ALL ")
query := fmt.Sprintf(`UPDATE %s t JOIN (%s) u ON t.id = u.id SET t.count = t.count + u.delta`, tbName, subQuery)
_, err = MASTER.ExecContext(ctx, query, args...)
```

### Insert On Duplicate

```go
query := fmt.Sprintf(`INSERT INTO %s (%v) VALUES (%v) ON DUPLICATE KEY UPDATE count = VALUES(count)`,
    tbName, insertFields, valuesFields)
_, err = MASTER.NamedExecContext(ctx, query, items)
```

### Transaction

```go
err = gutil.Trans(dao.MASTER, func(tx *sqlx.Tx) (err error) {
    err = dao.XXXX.Create(ctx, tx, item)
    return
})
```

### DAO with db.Search（列表查询）

```go
func (d *XXXXAdminDao) GetList(ctx context.Context, req *model.GetListXXXXAdminReq) (list []*model.XXXX, total int64, err error) {
    query := fmt.Sprintf(`SELECT * FROM %s`, model.TableNameXXXX)
    searchReq := db.BuildSearchReq(req.Page, req.PageSize).Sort("id", db.OrderTypeDESC).EnableCountTotal()
    if req.FilterUID > 0 { searchReq.Equal("uid", req.FilterUID) }
    total, err = db.Search(SLAVE.Unsafe(), &list, query, searchReq)
    return
}
```

---

## 
