# AllIn — Claude 项目说明

## 项目结构

```
allin/
├── allin-server/   # Go 后端（net/http, gorilla/websocket, MySQL）
├── allin-web/      # 前端（Vite + React + TypeScript + PixiJS v8）
└── docs/           # 设计文档和进度追踪
```

## 本地开发启动

### 前置条件
- MySQL 8.0 运行在 `127.0.0.1:13306`，root 无密码
- 连接命令：`mysql -h 127.0.0.1 -P 13306 -u root`
- 数据库 `allin` 已存在，启动时 AutoMigrate 自动建表

### 后端（allin-server/）
```bash
go run ./cmd/server
```
- 监听 `:8080`
- 健康检查：`curl http://localhost:8080/health`

### 前端（allin-web/）
```bash
npm run dev
```
- 默认 `:5173`，端口占用时顺延
- `/api` 代理到 `http://localhost:8080`（含 WebSocket）

### 验证后端是否已在运行
```bash
curl http://localhost:8080/health
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.25 |
| HTTP/WS | `net/http` + `gorilla/websocket` |
| 数据库 | MySQL 8.0，`go-sql-driver/mysql` |
| 鉴权 | JWT HS256，7 天有效期 |
| 前端构建 | Vite 5 + TypeScript |
| UI | React 18 + CSS Modules |
| 2D 渲染 | PixiJS v8（WebGL，程序化渲染，无外部图片） |
| 状态管理 | Zustand |
| 路由 | React Router v6 |

## 关键设计决策

- **无循环依赖**：`game` 包导入 `ws`；`ws` 不导入 `game`。通过 `ws.Handler.SetEngineStarter(func)` 回调在 `main.go` 中注入 Engine 工厂。
- **游戏引擎**：单 goroutine，`select` 监听 `hub.Inbound` + 可重置 timer channel（`nil` channel 不触发 select）。
- **座位旋转**：PixiJS TableScene 将本地玩家始终旋转到底部（display index 0）。
- **手牌评估**：纯 Go 枚举 C(7,5)=21 种五牌组合，`HandRanks.dat` 为可选加速路径。
- **边底池**：按 TotalBet 升序迭代，逐层计算各玩家贡献。

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
详见 `docs/PROGRESS.md`。当前阶段：Phase 8（UI 视觉重绘）进行中。

## Docs Conventions

- 文档存放目录：`docs`

- **SQL 文件**：所有 DDL 统一存放在 `docs/sql/`

- **SQL 备份文件**：如果我要求从本地数据使用 mysqldump 导出数据库备份文件，路径使用： `docs/sql/bak`，导出命令为：

  ~~~
  mysqldump --databases <数据库> --set-gtid-purged=OFF \
  --host=<主机> \
  --port=<端口> -u<用户名> -p<密码>
  ~~~

## Infrastructure

### MySQL

- Host: `127.0.0.1:13306` / User: `root` / Password: (empty)
- `shopman_base` — 基础业务库；`shopman_admin` — 管理后台库

```bash
mysql -h 127.0.0.1 -P 13306 -u root < docs/sql/shopman_base.sql
mysql -h 127.0.0.1 -P 13306 -u root < docs/sql/shopman_admin.sql
```

### Redis

- Addr: `127.0.0.1:7000` / Password: (empty)（Cluster 模式）

### Kafka

- Brokers: `127.0.0.1:9092`

### Config

- 假设微服务的名字为：base
- 微服务配置文件: `base/local.yaml`（viper 加载，字段见 `base/biz/model/config.go`）

---



## Data Model Conventions

- **Struct Tags**: 只用 `json` tag，不用 `db` tag

- **ID 类型**: 大多数用 `int64`；PostID 用 `uint64`，tag 为 `json:"post_id,string"`

- **审计字段**: `updater`, `created_at`, `updated_at`, `deleted_at`

- **软删除**: 更新 `deleted_at`，不做硬删除

- **错误处理**: `fmt.Errorf("xxx failed: %w", err)`

- **错误不忽略**: 禁止用 `_` 丢弃 `error`；确实无需处理时必须用 `util.Zerolog` 记录 Warn/Error 日志，例如：

  ```go
  if err := foo(); err != nil {
      util.Zerolog.Error().Err(err).Msg("foo failed")
  }
  // 后台 goroutine 中同理，不得写 _ = foo()
  ```

- **日志**: `util.Zerolog`

- **响应**: `gresp.FmtRsp(c, err, resp)`；参数错误用 `gresp.FmtParamInvalidRsp(c, err)`

- **Admin 命名**: Handler/Service/DAO 均加 `XXXXAdmin` 后缀

- **DB 枚举类型规范**：数据库表中所有 tinyint/int 枚举列，必须在 `gmodel/` 中定义对应的具名类型（`type XxxStatus int`），并遵守以下规则：

  - 枚举值**从 1 开始**，0 保留为 `XxxUnknown = 0`（表示未知/未设置）
  - 结构体字段类型使用具名枚举类型，不用裸 `int`
  - 例外：`UserSource`（JWT 元数据，App=0 为历史设计，不改动）

  ```go
  type OrderStatus int
  
  const (
      OrderStatusUnknown OrderStatus = 0 // 未知
      OrderStatusPending OrderStatus = 1 // 待处理
      OrderStatusDone    OrderStatus = 2 // 已完成
  )
  
  type Order struct {
      Status OrderStatus `json:"status"` // 状态：1=待处理 2=已完成
  }
  ```

- **DB 结构体字段注释**: 对应数据库表的结构体（`gmodel/` 下），每个字段必须加行尾注释说明含义，例如：

  ```go
  type Foo struct {
      ID        int64  `json:"id,string"`  // 主键
      UID       int64  `json:"uid"` // 用户ID
      CreatedAt int64  `json:"created_at"` // 创建时间（Unix 秒）
  }
  ```

---

## Code Patterns

### Common rules

- 代码简洁清晰，如果逻辑过于复杂则增加一下注释；

### Layer Service Template

```go
type XXXXSvc struct {}

func InitXXXXSvc() *XXXXSvc { return &XXXXSvc{} }

func (svc *XXXXSvc) XXXX(ctx context.Context, req *model.XXXXReq) (resp *model.XXXXResp, err error) {
    resp = new(model.XXXXResp)
    return
}
```

### Ticker Refresh Config

> 通用配置数据，使用本地内存存放，定时刷新

```go
type PostSettingSvc struct {
    Conf gsafe.Obj[*model.PostSettingCfg]
}

func InitPostSettingSvc() *PostSettingSvc {
    svc := &PostSettingSvc{}
    _ = svc.tickerUpdateConf(context.Background())
    go func() {
        tick := time.NewTicker(2 * time.Minute)
        defer tick.Stop()
        for range tick.C {
            _ = svc.tickerUpdateConf(context.Background())
        }
    }()
    return svc
}
func (svc *PostSettingSvc) tickerUpdateConf(ctx context.Context) (err error) {
	content := general_conf.GetBackendConfig("post_setting_cfg")
	tmpConf := &model.PostSettingCfg{}
	err = json.Unmarshal([]byte(content), tmpConf)
	if err != nil {
		err = fmt.Errorf("json.Unmarshal failed: %v", err)
		return
	}
	if tmpConf.MarkBoldThreshold == 0 {
		err = fmt.Errorf("mark_bold_threshold is zero")
		return
	}
	svc.Conf.Store(&tmpConf)
	return
}
```

### Local Cache Pattern

> local cache → gclt → redis/db

```go
func (c *XXXXCache) Get(ctx context.Context, ids []int64) (retMap map[int64]*gmodel.XXXX, err error) {
    retMap = map[int64]*gmodel.XXXX{}
    var notIDs []int64
    for _, id := range ids {
        if row := c.get(id); row != nil {
            retMap[id] = row
        } else {
            notIDs = append(notIDs, id)
        }
    }
    if len(notIDs) > 0 {
        list, _ := dao.XXXX.GetByIds(ctx, notIDs)
        for _, item := range list {
            retMap[item.ID] = item
            c.Set(item.ID, item)
        }
    }
    return
}
```

### gclt RPC Client

```go
func XXXXX(ctx context.Context, req *gmodel.XXXXXReq) (data *gmodel.XXXXXResp, err error) {
    resp := struct {
        gmodel.BaseResponse
        Data *gmodel.XXXXXResp `json:"data"`
    }{}
    reqBody, _ := json.Marshal(req)
    httpCode, err := PostBySvc(ctx, XXXXService, "/path/to/api", reqBody, &resp)
    if err != nil { return }
    if httpCode != consts.StatusOK { return nil, fmt.Errorf("httpCode: %d", httpCode) }
    if resp.Code != gmodel.CodeOK { return nil, fmt.Errorf("code: %d, msg: %s", resp.Code, resp.Message) }
    if resp.Data == nil { return nil, fmt.Errorf("data is nil") }
    return resp.Data, nil
}
```

---

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

## Admin Backend CRUD

### Model

```go
type (
    GetListXXXXAdminReq struct {
        gmodel.BaseArgs
        gmodel.PagePagination
        FilterUID int64 `query:"filter_uid"`
    }
    GetListXXXXAdminResp struct {
        Total int64         `json:"total"`
        List  []*model.XXXX `json:"list"`
    }
    EditXXXXAdminReq struct {
        gmodel.BaseArgs
        *model.XXXX
    }
)
```

### Router

```go
XXXX := admin.Group("/module")
{
    XXXX.GET("/cfg/list", handler.XXXXAdmin.GetListXXXX)
    XXXX.POST("/cfg/create", handler.XXXXAdmin.CreateXXXX)
    XXXX.POST("/cfg/update", handler.XXXXAdmin.UpdateXXXX)
    XXXX.POST("/cfg/delete", handler.XXXXAdmin.DeleteXXXX)
}
```

### Handler

```go
func (h *XXXXAdminHandler) GetListXXXX(ctx context.Context, c *app.RequestContext) {
    req := &model.GetListXXXXAdminReq{}
    if err := c.BindAndValidate(req); err != nil {
        gresp.FmtParamInvalidRsp(c, err)
        return
    }
    resp, err := service.XXXXAdmin.GetListXXXX(ctx, req)
    gresp.FmtRsp(c, err, resp)
}
```

### Service

```go
func (svc *XXXXAdminSvc) CreateXXXX(ctx context.Context, req *model.EditXXXXAdminReq) (resp *model.EditXXXXAdminResp, err error) {
    resp = new(model.EditXXXXAdminResp)
    ts := gutil.GetNow()
    req.CreatedAt, req.UpdatedAt, req.Updater = ts, ts, req.UID
    err = dao.XXXXAdmin.Create(ctx, req.XXXX)
    if err != nil { err = fmt.Errorf("create failed: %w", err) }
    return
}
```

