> https://stitch.withgoogle.com/projects/4278287175556993233?pli=1

# Design System: Galactic Casino

## 1. 概览

**主题定位：深空豪赌厅**

整体风格为深色太空 + 奢华赌场的融合：以极深的海军蓝/炭黑为背景，金色（#d4af37）作为核心强调色，配合玻璃态（Glassmorphism）浮层和绿色绒面牌桌，营造出漂浮于星云边缘的高级赌局氛围。

---

## 2. 色彩系统

所有颜色常量定义于 `ui-web/src/pixi/config/colors.ts`（PixiJS 侧）和各 CSS Module（React 侧）。

### 表面层级

| 变量名 | 十六进制 | 用途 |
|---|---|---|
| `VOID` | `#060c14` | 最深背景（游戏房间根容器） |
| `SURFACE_DIM` | `#10141a` | 全局 body 背景 |
| `SURFACE` | `#1c2026` | 卡片容器（大厅 section） |
| `SURFACE_HIGH` | `#262a31` | 抬升表面 |
| `SURFACE_BRIGHT` | `#353940` | 激活元素、tooltip |
| `GLASS` | `#040810` | 玻璃面板基色（配合 opacity） |

### 牌桌专用色

| 变量名 | 十六进制 | 用途 |
|---|---|---|
| `WOOD_FRAME` | `#2c1a08` | 深红木边框（桌边 rail） |
| `FELT_CENTER` | `#1e7a40` | 绒面中心（最亮绿） |
| `FELT_EDGE` | `#0c3320` | 绒面边缘（最暗绿） |

### 金色调

| 变量名 | 十六进制 | 用途 |
|---|---|---|
| `GOLD` | `#d4af37` | 主金色，贯穿全 UI |
| `GOLD_LIGHT` | `#f2ca50` | 超新星金，高光 / 登录页标题 |
| `GOLD_DIM` | `#b8952c` | 暗金，渐变尾色 / 阴影 |
| `GOLD_CONTAINER` | `#e9c349` | 金色表面色调 |

### 文字色

| 变量名 | 十六进制 | 用途 |
|---|---|---|
| `TEXT_PRIMARY` | `#dfe2eb` | 主要文字（亮白偏蓝） |
| `TEXT_SECONDARY` | `#d0c5af` | 次要文字 |
| `TEXT_DIM` | `#99907c` | 弱化 / 占位文字 |

### 功能色

| 颜色 | 十六进制 | 用途 |
|---|---|---|
| 绿色强调 | `#afcdbd` | 次要标签、等待提示文字 |
| 跟注绿 | `#4ade80` | Call 按钮及 hover 光晕 |
| 弃牌红 | `#f87171` | Fold 按钮 |
| 错误红 | `#ff5252` | 状态错误 |
| 警示青绿 | `#00c896` | 倒计时中段 |
| 断线红 | `#93000a → #b71c1c` | ConnectionBanner 渐变 |

---

## 3. 字体

双字体系统，定义于 `colors.ts`。

| 字体 | 变量 | 用途 |
|---|---|---|
| **Space Grotesk** | `FONT_HEADLINE` | 标题、数字、下注金额、标签、All-In 等高冲击场景 |
| **Manrope** | `FONT_BODY` | 玩家名字、说明文字、操作按钮标签、聊天内容 |
| **Inter** | 全局 body fallback | `index.css` 根元素默认字体 |

### 典型字重与间距

- Logo / 品牌名：`font-weight: 900`, `letter-spacing: 4px`, `font-style: italic`
- 操作按钮标签：`font-weight: 700`, `text-transform: uppercase`, `letter-spacing: 0.05em`
- 金额数字（Space Grotesk）：`font-weight: 800–900`
- 组标签（大厅表单）：`font-size: 0.7rem`, `font-weight: 700`, `letter-spacing: 0.12em`

---

## 4. 层叠与玻璃态

浮层 HUD 元素统一使用 Glassmorphism：

```css
background: rgba(4, 8, 16, 0.85–0.95);
backdrop-filter: blur(12–24px);
border: 1px solid rgba(212, 175, 55, 0.1–0.2);
border-radius: 12–20px;
```

层级（z-index）：

| 层级 | z-index | 元素 |
|---|---|---|
| CRT 扫描线叠加 | 100 | `RoomPage::before` |
| 暗角叠加 | 99 | `RoomPage::after` |
| 操作面板 / 聊天 / 历史 | 20–50 | `.overlay`, `.root` |
| ConnectionBanner | 999 | 全局顶部红色条 |

---

## 5. 特效

### CRT 扫描线（RoomPage）

游戏房间根容器上叠加两层伪元素：

- `::before`：扫描线（`background-size: 100% 3px`）+ 轻微 RGB 色散，`opacity: 0.25`
- `::after`：四角暗角晕影，`inset 0 0 150px rgba(0,0,0,0.8)`

### All-In 脉冲金光

```css
@keyframes pulseGold {
  0%, 100% { box-shadow: 0 0 12px rgba(212, 175, 55, 0.35); }
  50%       { box-shadow: 0 0 26px rgba(212, 175, 55, 0.65); }
}
```

### 手牌结算浮层入场

```css
@keyframes fadeIn {
  from { opacity: 0; transform: translate(-50%, -48%); }
  to   { opacity: 1; transform: translate(-50%, -50%); }
}
```

### 断线 Banner 滑入

```css
@keyframes slideDown {
  from { transform: translateY(-100%); }
  to   { transform: translateY(0); }
}
```

---

## 6. PixiJS 画布布局

画布固定分辨率 **1600 × 900**（16:9），定义于 `ui-web/src/pixi/config/layout.ts`。

### 牌桌椭圆

| 参数 | 值 |
|---|---|
| 中心 | (800, 385) |
| X 半轴（绒面外沿） | 555px |
| Y 半轴（绒面外沿） | 220px |
| 木质边框厚度 | 17px |
| 圆角半径（九宫格纹理） | 280px |

### 座位轨道

9 个座位沿椭圆轨道分布（A=580, B=262），极坐标生成：

```
         [5]   [4]   [3]
     [6]                 [2]
     [7]                 [1]
         [8]       [0←你]
```

- 本地玩家（座位 0）固定在底部，头像半径 **44px**（直径 88px）
- 远端玩家头像半径 **30px**（直径 60px）

### 扑克牌尺寸

| 类型 | 宽 | 高 | 圆角 |
|---|---|---|---|
| 公共牌（桌面） | 64px | 90px | 10px |
| 手牌（座位旁） | 46px | 64px | 7px |

### 扑克牌配色

- 正面：奶白 `#faf8f0`，普通边框 `rgba(200,200,200,0.7)`，高亮边框 `#d4af37` + 金色光晕
- 背面：深蓝 `#1a2a5e`，斜纹图案 `rgba(255,255,255,0.04)`
- 红色花色（♥♦）：`#c0392b`
- 黑色花色（♣♠）：`#1a1a2e`

---

## 7. 组件规范

### 顶部导航栏（RoomPage）

```css
background: rgba(4, 8, 16, 0.9);
backdrop-filter: blur(12px);
border-bottom: 1px solid rgba(212, 175, 55, 0.1);
```

- 品牌名：Space Grotesk, 900, italic, `#d4af37`, letter-spacing: -0.02em
- 导航链接：Space Grotesk, 0.7rem, uppercase, letter-spacing: 0.2em，hover 变金色
- 当前街道徽章（PREFLOP/FLOP…）：Space Grotesk, 0.85rem, letter-spacing: 3px, `#d4af37`

### 操作 Dock（ActionPanel）

```css
background: rgba(4, 8, 16, 0.92);
backdrop-filter: blur(24px);
border-radius: 20px;
border: 1px solid rgba(212, 175, 55, 0.18);
box-shadow: 0 -12px 40px rgba(0,0,0,0.7), inset 0 1px 0 rgba(255,255,255,0.04);
```

按钮语义色：

| 按钮 | 颜色方案 |
|---|---|
| Fold（弃牌） | 红色 `#f87171`，hover 有红色背景 + 边框 |
| Check（过牌） | 中性灰白 `#e2e8f0`，低调 |
| Call（跟注） | 绿色 `#4ade80`，hover 有绿色光晕 |
| Raise（加注组） | 金色边框区域，内含滑条 + 预设比例按钮 |
| All-In | 金色渐变 `#d4af37 → #b8952c`，持续脉冲动画 |

加注滑条：金色填充 track，`#d4af37` 圆形 thumb，active 时 1.25x 缩放 + 强光晕。

### 聊天面板（ChatPanel）

- 位置：左下角，绝对定位，`bottom: 80px, left: 16px`
- 宽度：收起 260px，展开 300px
- 发送按钮：金色渐变 `135deg, #d4af37, #b8952c`，hover 金色光晕

### 手牌结算浮层（HandHistory）

- 居中覆盖，`backdrop-filter: blur(16px)`
- 金色边框 `rgba(212,175,55,0.4)`，外发光 `0 0 40px rgba(212,175,55,0.15)`
- 赢额颜色：`#afcdbd`（次要绿）

### 登录页（LoginPage）

- 背景：`#0f1923`
- 卡片：`#1a2535`，`border-radius: 16px`，`box-shadow: 0 8px 32px rgba(0,0,0,0.5)`
- 标题：`#f0c040`，2.4rem，800，letter-spacing: 4px
- 激活 tab：`background: #f0c040`，文字 `#0f1923`
- 主按钮：`background: #f0c040`，文字 `#0f1923`，hover 降低 opacity

### 大厅页（LobbyPage）

- 背景：`#0a0f18`
- Header：`#111820` + 底部金色分割线 `rgba(212,175,55,0.12)`
- 筹码余额徽章：金色边框 pill，背景 `rgba(212,175,55,0.1)`
- Section 卡片：`#111820`，`border-radius: 12px`，`border: 1px solid rgba(255,255,255,0.06)`
- 组标签（盲注/买入等）：`rgba(212,175,55,0.6)`，大写，极小字号
- 金额输入框：深底色 + 金色 `$` 前缀块，focus 时边框变 `#d4af37`
- 主操作按钮：`#d4af37`，dark 文字，hover 微上移 `-1px`

### 断线提示条（ConnectionBanner）

```css
background: linear-gradient(135deg, #93000a, #b71c1c);
color: #ffdad6;
animation: slideDown 0.2s ease;
```

---

## 7.5 PixiJS 组件视觉规范

### 座位精灵（SeatSprite）

两种模式，视觉结构不同：

| 元素 | 本地玩家（底部大头像） | 远端玩家 |
|---|---|---|
| 头像半径 | 44px（直径 88px） | 30px（直径 60px） |
| 名称样式 | 金色底板（nameBadge）+ 黑字 | 头像下方白色文字 |
| 筹码面板 | 宽体玻璃底板 + 金色边框 | 名称下方紧凑显示 |
| 手牌位置 | 扇形展开于右侧 | 显示在头像上方 |

**头像视觉层次（从底到顶）：**
1. `breathGlow` — 空座位呼吸光晕（sin 缓入缓出，吸引入座）
2. `avatarBg` — 圆形底板 + 描边
3. `allInGlow` — ALL-IN 脉冲金色光晕（sin 驱动）
4. `avatarImg` — 头像图片（圆形裁切遮罩）
5. `posTag` — 位置标签（BTN/SB/BB/UTG），悬浮于头像 225° 方向
6. `betBadge` — 下注徽章，朝牌桌中心浮动（绿底 + 筹码图标 + 金额）
7. `statusBadge` — 状态徽章："已弃牌"（灰）/ "ALL-IN"（金）
8. `card0`, `card1` — 两张手牌 CardSprite

### 倒计时圆弧（TimerArc）

覆盖在当前行动玩家头像外沿 +4px 处，每帧 tick 更新。

**三段变色（剩余时间比例）：**

| 阶段 | 条件 | 颜色 |
|---|---|---|
| 充裕 | > 50% | `TEXT_PRIMARY` `#dfe2eb`（银白） |
| 警示 | 15%–50% | `WARN` `#00c896`（青绿） |
| 紧迫 | ≤ 15% | `ERROR` `#ff5252`（红） |

**绘制方式：** 外发光层（宽 7px，alpha 0.15）+ 实线层（宽 3px，alpha 0.90）双层叠加。底环为暗金色 `GOLD_DIM`，alpha 0.15。右上角秒数徽章：玻璃底圆角矩形 + 彩色描边，Space Grotesk 金色文字。

最后 10 秒每整秒播放 tick 音效。

### 筹码精灵（CasinoChip）

程序化绘制，10 层视觉结构：

> 阴影 → 金属外环（`#c8c8d4`）→ 主体色 → 8 个边缘卡槽（每 45°）→ 压边细环 → 24 齿装饰环 → 深色中心区 → 高光 → 面值文字 → 轮廓描边

**面额 → 外观映射：**

| 范围 | 主色 | 卡槽色 | 面值标签 |
|---|---|---|---|
| ≤ $4 | `#dcdce8`（灰白） | `#888899` | $1 |
| ≤ $24 | `#c0231e`（红） | `#ffffff` | $5 |
| ≤ $99 | `#1a7a3a`（绿） | `#ffffff` | $25 |
| ≤ $499 | `#1a1a1a`（黑） | `#d4af37`（金） | $100 |
| ≤ $999 | `#5a1a8a`（紫） | `#ffffff` | $500 |
| $1000+ | `#c07800`（橙金） | `#1a1a1a` | $1K |

面值文字字体：Georgia serif，`font-weight: 900`，颜色根据主色亮度自动切换黑/白。

### 底池显示（PotDisplay）

药丸形组件（200×36px），固定居中在公共牌下方。

**视觉结构（左→右）：**
1. 多层金色晕光（6 层同心 roundRect，扩散 2–28px，alpha 0.025–0.18）
2. 玻璃底板（`GLASS` + alpha 0.8）+ 金色描边（alpha 0.4）
3. 左侧装饰筹码堆（3 列侧视角，红/青/金，静态装饰）
4. "POT" 标签（Space Grotesk，`GOLD_DIM`）
5. 金额数字（Space Grotesk 900，`GOLD`，`$1,200` 格式）

装饰筹码堆采用椭圆透视压缩（chipW=10, chipH=4），每枚 4 层：厚度层 + 顶面 + 轮廓 + 白色 decal，模拟 3D 侧视角堆叠。

---

## 8. 设计规则

### 必须遵守

- 背景色必须使用色板中的深色（最深 `#060c14`），禁止使用纯 `#000000`
- 浮层 HUD 必须使用 `backdrop-filter: blur(12px+)` 保持玻璃感
- 金色用于所有"价值"信息：筹码、下注金额、当前状态、焦点边框
- 圆角范围：`8px`（小元素）到 `20px`（大容器），不使用尖角
- 按钮 hover 使用 `opacity` 或轻微 `scale(1.04-1.08)` + 光晕，不用颜色突变

### 禁止

- 布局分割线禁止用 `1px solid` 实线（导航栏底边除外）；用背景色差或 padding 区分
- 阴影禁止使用 Material 风格重投影；使用低透明度大范围环境光晕
- 聊天 / 历史等文字区域禁止使用分割线，用 `gap` 间距