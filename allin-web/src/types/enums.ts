// 前端统一枚举定义

/** 连接状态 */
export enum ConnectionStatus {
    Connected = 'connected',       // WebSocket 已连接
    Reconnecting = 'reconnecting', // 断线重连中
    Disconnected = 'disconnected', // 已断开连接
}

/** 牌局阶段 */
export enum Street {
    Idle = 'idle',           // 等待中，未开始
    PreFlop = 'preflop',     // 翻牌前（发手牌后）
    Flop = 'flop',           // 翻牌（3张公共牌）
    Turn = 'turn',           // 转牌（第4张公共牌）
    River = 'river',         // 河牌（第5张公共牌）
    Showdown = 'showdown',   // 摊牌
}

/** 玩家操作 */
export enum PlayerAction {
    Fold = 'fold',       // 弃牌
    Check = 'check',     // 过牌（不加注）
    Call = 'call',       // 跟注
    Bet = 'bet',         // 下注（本轮首次）
    Raise = 'raise',     // 加注
    AllIn = 'all_in',    // 全押
}

/** WebSocket 事件类型（服务端→客户端 & 客户端→服务端） */
export enum WSEventType {
    // ── 服务端 → 客户端（下行） ──
    Connected = 'connected',           // 连接成功，返回座位快照
    PlayerJoined = 'player_joined',    // 有玩家加入房间
    PlayerLeft = 'player_left',        // 有玩家离开房间
    GameStarted = 'game_started',      // 新一手牌开始
    HoleCards = 'hole_cards',          // 发送本人手牌（仅发给当事人）
    CardsDealt = 'cards_dealt',        // 公共牌发出（flop/turn/river）
    StreetStarted = 'street_started',  // 新街道开始，附带底池等信息
    ActionRequired = 'action_required', // 通知某玩家需要行动
    ActionTaken = 'action_taken',      // 广播某玩家已执行的操作
    ActionTimeout = 'action_timeout',  // 玩家超时，服务端自动处理
    Showdown = 'showdown',             // 摊牌阶段，公开各玩家手牌
    HandResult = 'hand_result',        // 本手结算，包含赢家和筹码变动
    ChatMessage = 'chat_message',      // 聊天消息（双向）
    // ── 客户端 → 服务端（上行） ──
    JoinRoom = 'join_room',            // 请求加入房间
    Action = 'action',                 // 提交玩家操作（fold/check/call/bet/raise/all_in）
}

/** 登录/注册模式 */
export enum AuthMode {
    Login = 'login',       // 登录
    Register = 'register', // 注册
}

/** AI 风格 */
export enum BotStyle {
    Mixed = 'mixed',           // 混合风格（随机切换）
    Aggressive = 'aggressive', // 激进型（高频加注）
    Passive = 'passive',       // 被动型（倾向跟注/过牌）
    Random = 'random',         // 随机行为
}

/** WebSocket 内部事件（前端内部使用，不发送到服务端） */
export enum WSInternalEvent {
    Open = '__open__',   // 连接建立
    Close = '__close__', // 连接关闭
    Error = '__error__', // 连接异常
}
