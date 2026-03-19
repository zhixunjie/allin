> https://github.com/zhixunjie/allin
>
> refer：
>
> https://github.com/search?q=%E5%BE%B7%E5%B7%9E%E6%89%91%E5%85%8B&type=repositories
>
> https://www.waliyouxi.com/demo/app
>
> https://www.sud.tech/cn/gamelist

我的想法：

- 技术栈：
  - Go + React + PixiJS
  - 消息协议：WebSocket + JSON/Protobuf
- 架构设计：
  - 规则引擎：
    - State Machine（控制游戏状态）
    - 比牌逻辑（Two Plus Two算法）
- 产品定位：
  - 前期先做 PC Web，后面在扩展到 移动端H5 / Android / IOS
  - 组局形式：熟人组局

视觉设计规范：

~~~
# 星空德州 (Galactic Aces) 视觉设计规范 v4.0

## 1. 核心视觉理念：奢华科幻博弈 (Luxury Sci-fi Poker)
**设计灵魂**：将顶级赌场的经典写实质感（深绿绒布、实木边框、标准纸质扑克）置于深邃、神秘的星空背景之中。这是一种“星际间的皇家赌场”体验。

---

## 2. 颜色与材质数值 (Color & Texture)

### 2.1 环境与背景
*   **主背景 (Canvas)**: `0x060c14` (近黑深蓝，代表宇宙深处)。
*   **星空特效**: 随机散布的白色星点 (Alpha: 0.05 ~ 0.35) + 极微弱的流动星云。
*   **页头栏 (Header)**: `0x040810` (高对比深蓝，增加结构感)。

### 2.2 顶级赌场牌桌
*   **实木边框 (Wooden Frame)**: `0x2c1a08`。采用4层叠加工艺 (Alpha: 0.9 / 0.7 / 0.5 / 0.3)，模拟厚重实木的质感。
*   **金色饰线 (Golden Rim)**: `0xd4af37` (Alpha: 0.6)。带有 1px-3px 的外发光效果，代表星际间的奢华点缀。
*   **深绿绒布 (Green Felt)**: 采用径向渐变。
    *   中心高亮: `0x1e7a40` (Alpha: 0.3)
    *   边缘基底: `0x0c3320` (Alpha: 1.0)

---

## 3. 游戏核心资产 (Game Assets)

### 3.1 扑克牌 (Casino Standard)
*   **正面**: 纯白纸质底色 (0xfaf8f0)，标准红 (#ff0000)/黑 (#000000) 花色，经典衬线字体。
*   **背面**: 经典深红或深蓝，带有传统、精致的几何花纹。
*   **动效**: 3D 轴向翻转 + `backOut` 缓动。

### 3.2 筹码 (Clay Texture)
*   **风格**: 具有写实粘土质感的圆形彩色筹码（红、蓝、绿、黑、黄）。
*   **动态展示**: 下注区域根据数额动态展示筹码堆高度，增强博弈实感。

---

## 4. 界面布局与交互 (Layout & Interaction)

### 4.1 垂直分布结构
*   **行动栏居中**: 底部交互按钮水平绝对居中。
*   **呼吸间距**: 牌桌底部 -> 手牌强度条 -> 行动展示栏 之间保持 **24px - 32px** 的间隙，减少底部留白。

### 4.2 个人状态 (Seat 0)
*   **手牌位置**: 头像右侧，自然重叠。
*   **筹码余额**: 独立展示，确保不被手牌或强度条遮挡。

### 4.3 动态反馈
*   **行动位置标签**: 金色胶囊形状 (`0xd4af37` 边框)，半透明背景。包括 SB/BB/UTG/HJ/CO/BTN 等。
*   **倒计时**: 玩家头像外围金色环形进度条 + 剩余秒数。
*   **手牌强度**: 根据牌型等级改变条形图的颜色饱和度与发光频率。
*   **结算特效**: 最佳5张牌高亮上浮 20px + 金色粒子喷发（带重力感）。

---

## 5. 字体规范
*   **数字/品牌**: `Space Grotesk` (大写，宽间距)。
*   **交互文本**: `Inter` 或 `PingFang SC` (简体中文，清晰易读)。
~~~

代码：

~~~
<!DOCTYPE html>

<html class="dark" lang="zh-CN"><head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<title>GALACTIC ACES - 高额德州扑克</title>
<script src="https://cdn.tailwindcss.com?plugins=forms,container-queries"></script>
<link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300;400;500;700;900&amp;family=Manrope:wght@300;400;600;800&amp;display=swap" rel="stylesheet"/>
<link href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:wght,FILL@100..700,0..1&amp;display=swap" rel="stylesheet"/>
<script id="tailwind-config">
        tailwind.config = {
            darkMode: "class",
            theme: {
                extend: {
                    colors: {
                        "primary-fixed-dim": "#e9c349",
                        "surface-container-lowest": "#0a0e14",
                        "secondary-fixed": "#cbead8",
                        "background": "#060c14",
                        "inverse-surface": "#dfe2eb",
                        "outline": "#99907c",
                        "primary-fixed": "#ffe088",
                        "on-tertiary-container": "#6f3329",
                        "primary": "#d4af37",
                        "on-error-container": "#ffdad6",
                        "on-primary-fixed-variant": "#574500",
                        "tertiary-fixed-dim": "#ffb4a7",
                        "on-background": "#dfe2eb",
                        "on-secondary-container": "#a1bfaf",
                        "error": "#ffb4ab",
                        "tertiary-container": "#f19d8f",
                        "on-error": "#690005",
                        "on-tertiary": "#551f17",
                        "secondary": "#afcdbd",
                        "surface-dim": "#10141a",
                        "on-tertiary-fixed-variant": "#71352b",
                        "surface": "#10141a",
                        "surface-container-high": "#262a31",
                        "secondary-container": "#0c3320",
                        "primary-container": "#d4af37",
                        "on-primary-container": "#554300",
                        "on-surface-variant": "#d0c5af",
                        "secondary-fixed-dim": "#afcdbd",
                        "on-primary-fixed": "#241a00",
                        "on-secondary": "#1b352a",
                        "surface-bright": "#353940",
                        "inverse-on-surface": "#2d3137",
                        "surface-container": "#1c2026",
                        "on-secondary-fixed": "#052016",
                        "on-tertiary-fixed": "#390b05",
                        "inverse-primary": "#735c00",
                        "tertiary-fixed": "#ffdad4",
                        "surface-container-low": "#181c22",
                        "surface-variant": "#31353c",
                        "surface-tint": "#e9c349",
                        "error-container": "#93000a",
                        "outline-variant": "#4d4635",
                        "surface-container-highest": "#31353c",
                        "tertiary": "#ffbfb4",
                        "on-surface": "#dfe2eb",
                        "on-secondary-fixed-variant": "#314c40",
                        "on-primary": "#3c2f00"
                    },
                    fontFamily: {
                        "headline": ["Space Grotesk"],
                        "body": ["Manrope"],
                        "label": ["Manrope"]
                    },
                    borderRadius: {"DEFAULT": "0.25rem", "lg": "0.5rem", "xl": "0.75rem", "full": "9999px"},
                },
            },
        }
    </script>
<style>
        .material-symbols-outlined {
            font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
        }
        .starfield {
            background-color: #060c14;
            background-image: 
                radial-gradient(circle at 2px 2px, rgba(255, 255, 255, 0.15) 1px, transparent 0),
                radial-gradient(circle at 100px 100px, rgba(255, 255, 255, 0.05) 1px, transparent 0);
            background-size: 80px 80px, 150px 150px;
        }
        .crt-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: linear-gradient(rgba(18, 16, 16, 0) 50%, rgba(0, 0, 0, 0.1) 50%), linear-gradient(90deg, rgba(255, 0, 0, 0.03), rgba(0, 255, 0, 0.01), rgba(0, 0, 255, 0.03));
            background-size: 100% 3px, 3px 100%;
            pointer-events: none;
            z-index: 100;
            opacity: 0.3;
        }
        .vignette {
            position: fixed;
            inset: 0;
            box-shadow: inset 0 0 150px rgba(0,0,0,0.8);
            pointer-events: none;
            z-index: 99;
        }
        .glass-panel {
            background: rgba(4, 8, 16, 0.8);
            backdrop-filter: blur(12px);
        }
        .poker-card-container {
            perspective: 1000px;
            width: 5.5rem;
            height: 7.7rem;
        }
        .poker-card-inner {
            position: relative;
            width: 100%;
            height: 100%;
            text-align: center;
            transition: transform 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
            transform-style: preserve-3d;
        }
        .poker-card-container.is-flipped .poker-card-inner {
            transform: rotateY(180deg);
        }
        .poker-card-front, .poker-card-back {
            position: absolute;
            width: 100%;
            height: 100%;
            -webkit-backface-visibility: hidden;
            backface-visibility: hidden;
            border-radius: 0.625rem;
            box-shadow: 0 4px 12px rgba(0,0,0,0.4);
        }
        .poker-card-front {
            background: #faf8f0;
            border: 2px solid #d4af37;
            color: #1a1a2e;
            transform: rotateY(180deg);
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            padding: 0.5rem;
        }
        .poker-card-back {
            background: #0f3d24;
            border: 2px solid #d4af37;
            background-image: radial-gradient(circle at center, rgba(201, 168, 76, 0.2) 0%, transparent 70%),
                              repeating-linear-gradient(45deg, transparent, transparent 10px, rgba(201, 168, 76, 0.05) 10px, rgba(201, 168, 76, 0.05) 11px),
                              repeating-linear-gradient(-45deg, transparent, transparent 10px, rgba(201, 168, 76, 0.05) 10px, rgba(201, 168, 76, 0.05) 11px);
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .poker-card-back::after {
            content: '♦';
            color: rgba(201, 168, 76, 0.3);
            font-size: 2.5rem;
        }

        @keyframes ring-pulse {
            0% { filter: drop-shadow(0 0 2px rgba(212, 175, 55, 0.5)); opacity: 0.8; }
            50% { filter: drop-shadow(0 0 10px rgba(212, 175, 55, 0.8)); opacity: 1; }
            100% { filter: drop-shadow(0 0 2px rgba(212, 175, 55, 0.5)); opacity: 0.8; }
        }
        .animate-ring-pulse {
            animation: ring-pulse 2s ease-in-out infinite;
        }

        @keyframes breathe-gold {
            0%, 100% { box-shadow: 0 0 15px rgba(212, 175, 55, 0.3), inset 0 0 10px rgba(212, 175, 55, 0.1); }
            50% { box-shadow: 0 0 35px rgba(212, 175, 55, 0.7), inset 0 0 20px rgba(212, 175, 55, 0.3); }
        }
        .animate-breathe-gold {
            animation: breathe-gold 3s ease-in-out infinite;
        }

        @keyframes tag-glow {
            0%, 100% { box-shadow: 0 0 8px rgba(212, 175, 55, 0.3); border-color: rgba(212, 175, 55, 0.5); }
            50% { box-shadow: 0 0 20px rgba(212, 175, 55, 0.8); border-color: rgba(212, 175, 55, 1); }
        }
        .animate-tag-glow {
            animation: tag-glow 2s ease-in-out infinite;
        }

        @keyframes prismatic-gold {
            0% { filter: drop-shadow(0 0 5px #d4af37) hue-rotate(0deg); opacity: 0.8; }
            50% { filter: drop-shadow(0 0 20px #ffe088) hue-rotate(15deg); opacity: 1; }
            100% { filter: drop-shadow(0 0 5px #d4af37) hue-rotate(0deg); opacity: 0.8; }
        }
        .strength-bar-gold {
            background: linear-gradient(90deg, #1e7a40, #d4af37);
            box-shadow: 0 0 15px rgba(212, 175, 55, 0.6);
            animation: prismatic-gold 2s ease-in-out infinite;
        }

        .table-felt {
            background: radial-gradient(circle at center, #1e7a40 0%, #1a6b3a 40%, #145a32 65%, #0c3320 100%);
            box-shadow: inset 0 0 120px rgba(0,0,0,0.8);
        }
        .wooden-frame {
            background: #2c1a08;
            border-color: #2c1a08;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
        }
        .gold-border {
            border: 3px solid rgba(212, 175, 55, 0.6);
            box-shadow: 0 0 25px rgba(212, 175, 55, 0.3), inset 0 0 20px rgba(212, 175, 55, 0.2);
        }
        .inner-gold-line {
            border: 1.5px solid rgba(212, 175, 55, 0.12);
        }
        .pulse-gold {
            animation: pulse-gold 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
        @keyframes pulse-gold {
            0%, 100% { box-shadow: 0 0 15px rgba(212, 175, 55, 0.4); background-color: rgba(212, 175, 55, 0.1); }
            50% { box-shadow: 0 0 30px rgba(212, 175, 55, 0.7); background-color: rgba(212, 175, 55, 0.3); }
        }
        .raise-btn-hover:hover {
            transform: scale(1.1);
            box-shadow: 0 0 20px rgba(212, 175, 55, 0.6);
            background-color: rgba(212, 175, 55, 0.2);
            color: #fff;
        }

        .pos-tag {
            font-family: 'Space Grotesk', sans-serif;
            background: linear-gradient(180deg, rgba(20, 25, 35, 0.95) 0%, rgba(10, 14, 20, 0.98) 100%);
            border: 1px solid rgba(212, 175, 55, 0.6);
            color: #d4af37;
            font-size: 11px;
            font-weight: 800;
            padding: 2px 8px;
            border-radius: 9999px;
            text-transform: uppercase;
            letter-spacing: 0.12em;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
            z-index: 45;
            display: flex;
            align-items: center;
            justify-content: center;
            backdrop-filter: blur(4px);
        }

        .chip-stack {
            position: relative;
            width: 24px;
            height: 32px;
        }
        .chip {
            position: absolute;
            left: 0;
            width: 24px;
            height: 12px;
            background: #d4af37;
            border-radius: 50%;
            border: 1px solid rgba(0,0,0,0.3);
            box-shadow: 0 2px 0 #b8952c;
        }
    </style>
</head>
<body class="bg-background text-on-background font-body overflow-hidden h-screen w-screen starfield flex flex-col">
<div class="crt-overlay"></div>
<div class="vignette"></div>
<!-- TopAppBar -->
<header class="fixed top-0 w-full z-50 bg-[#040810]/90 backdrop-blur-md flex justify-between items-center px-6 py-4 border-b border-[#d4af37]/10">
<div class="flex items-center gap-8">
<span class="text-[#d4af37] font-black italic tracking-tighter text-2xl uppercase">GALACTIC ACES</span>
<div class="hidden md:flex gap-8 font-['Space_Grotesk'] uppercase tracking-[0.2em] text-xs">
<a class="text-[#d4af37] border-b-2 border-[#d4af37] py-1 transition-all duration-300" href="#">牌桌</a>
<a class="text-on-background/50 hover:text-[#d4af37] transition-all duration-300 py-1" href="#">大厅</a>
<a class="text-on-background/50 hover:text-[#d4af37] transition-all duration-300 py-1" href="#">锦标赛</a>
</div>
</div>
<div class="flex items-center gap-6">
<div class="flex flex-col items-end mr-4">
<span class="text-[10px] text-[#d4af37]/60 uppercase font-bold">牌桌 #089</span>
<span class="text-xs font-medium">盲注: 500/1000</span>
</div>
<button class="text-on-background/60 hover:text-[#d4af37] transition-colors duration-300">
<span class="material-symbols-outlined" data-icon="account_balance_wallet">account_balance_wallet</span>
</button>
<button class="text-on-background/60 hover:text-[#d4af37] transition-colors duration-300">
<span class="material-symbols-outlined" data-icon="history">history</span>
</button>
<button class="text-on-background/60 hover:text-[#d4af37] transition-colors duration-300">
<span class="material-symbols-outlined" data-icon="settings">settings</span>
</button>
</div>
</header>
<!-- Main Game Canvas -->
<main class="flex-grow relative w-full flex items-center justify-center p-8 pt-24 pb-32">
<!-- The Poker Table Container -->
<div class="relative w-full max-w-6xl aspect-[16/8] flex items-center justify-center z-10">
<!-- Table Shadow/Outer Glow -->
<div class="absolute inset-0 bg-[#d4af37]/5 blur-[120px] opacity-40 rounded-full"></div>
<!-- The Oval Table (Dark Wood Frame) -->
<div class="relative w-full h-full wooden-frame rounded-full border-[14px] border-[#2c1a08] overflow-hidden">
<!-- Prominent Glowing Gold Trim -->
<div class="absolute inset-0 gold-border rounded-full z-10 pointer-events-none"></div>
<!-- Inner Felt -->
<div class="absolute inset-0 table-felt flex items-center justify-center">
<!-- Inner Gold Decoration Lines -->
<div class="absolute inset-12 inner-gold-line rounded-full pointer-events-none opacity-50"></div>
<!-- Community Cards -->
<div class="flex flex-col items-center gap-8 z-10">
<div class="flex gap-4">
<!-- Community Card 1 -->
<div class="poker-card-container is-flipped">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front">
<div class="flex flex-col items-start leading-none">
<span class="text-[#c0392b] font-black text-xl">A</span>
<span class="text-[#c0392b] text-sm">♥</span>
</div>
<span class="self-center text-[#c0392b] font-black text-4xl">♥</span>
<div class="flex flex-col items-end leading-none rotate-180">
<span class="text-[#c0392b] font-black text-xl">A</span>
<span class="text-[#c0392b] text-sm">♥</span>
</div>
</div>
</div>
</div>
<!-- Community Card 2 -->
<div class="poker-card-container is-flipped">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front">
<div class="flex flex-col items-start leading-none">
<span class="text-[#c0392b] font-black text-xl">K</span>
<span class="text-[#c0392b] text-sm">♦</span>
</div>
<span class="self-center text-[#c0392b] font-black text-4xl">♦</span>
<div class="flex flex-col items-end leading-none rotate-180">
<span class="text-[#c0392b] font-black text-xl">K</span>
<span class="text-[#c0392b] text-sm">♦</span>
</div>
</div>
</div>
</div>
<!-- Community Card 3 -->
<div class="poker-card-container is-flipped">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front">
<div class="flex flex-col items-start leading-none">
<span class="text-[#1a1a2e] font-black text-xl">10</span>
<span class="text-[#1a1a2e] text-sm">♠</span>
</div>
<span class="self-center text-[#1a1a2e] font-black text-4xl">♠</span>
<div class="flex flex-col items-end leading-none rotate-180">
<span class="text-[#1a1a2e] font-black text-xl">10</span>
<span class="text-[#1a1a2e] text-sm">♠</span>
</div>
</div>
</div>
</div>
<!-- Community Card 4 (Back side) -->
<div class="poker-card-container">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front"></div>
</div>
</div>
<!-- Community Card 5 (Back side) -->
<div class="poker-card-container">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front"></div>
</div>
</div>
</div>
<!-- Pot Indicator with Visual Chip Stacks -->
<div class="flex items-center gap-5 px-10 py-4 bg-[#040810]/80 rounded-full border border-[#d4af37]/40 backdrop-blur-xl animate-breathe-gold">
<div class="flex gap-1.5 items-end">
<div class="chip-stack">
<div class="chip" style="bottom: 0px"></div>
<div class="chip" style="bottom: 4px"></div>
<div class="chip" style="bottom: 8px"></div>
<div class="chip" style="bottom: 12px"></div>
<div class="chip" style="bottom: 16px"></div>
</div>
<div class="chip-stack">
<div class="chip" style="bottom: 0px"></div>
<div class="chip" style="bottom: 4px"></div>
<div class="chip" style="bottom: 8px"></div>
<div class="chip" style="bottom: 12px"></div>
</div>
<div class="chip-stack">
<div class="chip" style="bottom: 0px"></div>
<div class="chip" style="bottom: 4px"></div>
<div class="chip" style="bottom: 8px"></div>
<div class="chip" style="bottom: 12px"></div>
<div class="chip" style="bottom: 16px"></div>
<div class="chip" style="bottom: 20px"></div>
</div>
</div>
<div class="flex flex-col">
<span class="text-[10px] text-[#d4af37]/60 font-black uppercase tracking-widest leading-none mb-1">当前底池</span>
<span class="font-headline text-[#d4af37] font-black tracking-[0.1em] text-2xl">$12,450</span>
</div>
</div>
</div>
</div>
</div>
<!-- Player Seats -->
<!-- Player 0 (Bottom/Center - User) -->
<div class="absolute left-1/2 -bottom-24 -translate-x-1/2 flex items-center z-40">
<div class="flex flex-col items-center">
<div class="w-32 h-32 rounded-full bg-surface-container-high shadow-[0_0_30px_rgba(212,175,55,0.4)] flex items-center justify-center relative ring-8 ring-[#d4af37]/10">
<span class="text-6xl z-10">👨‍🚀</span>
<div class="absolute inset-0 z-0 animate-ring-pulse">
<svg class="w-full h-full -rotate-90" viewbox="0 0 100 100">
<circle cx="50" cy="50" fill="none" r="46" stroke="rgba(212, 175, 55, 0.1)" stroke-width="4"></circle>
<circle cx="50" cy="50" fill="none" r="46" stroke="#d4af37" stroke-dasharray="289" stroke-dashoffset="72" stroke-linecap="round" stroke-width="4"></circle>
</svg>
</div>
<div class="absolute -top-1 -right-1 bg-[#040810] border border-[#d4af37] text-[#d4af37] text-[10px] font-black px-1.5 py-0.5 rounded-full shadow-[0_0_10px_rgba(212,175,55,0.4)] z-20 animate-pulse">15s</div>
<div class="absolute -bottom-2 w-max px-4 py-1.5 bg-[#d4af37] text-[#060c14] rounded-lg font-black text-xs shadow-lg uppercase tracking-widest z-20">你自己</div>
<div class="absolute -top-1 -left-1 pos-tag">BB</div>
</div>
<div class="bg-[#040810] px-8 py-3 rounded-xl border border-[#d4af37]/30 text-center shadow-2xl min-w-[140px] mt-2">
<p class="text-[#d4af37] font-headline font-black text-2xl">$45,000</p>
<p class="text-[9px] text-[#afcdbd] font-bold uppercase tracking-widest opacity-70">筹码余额</p>
</div>
</div>
<!-- User Hole Cards -->
<div class="ml-6 flex -space-x-10 translate-y-[-10px]">
<div class="poker-card-container is-flipped rotate-[-5deg] z-0">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front !p-2 !w-20 !h-28">
<div class="flex flex-col items-start leading-none scale-75 origin-top-left">
<span class="text-[#c0392b] font-black text-lg">A</span>
<span class="text-[#c0392b] text-xs">♥</span>
</div>
<span class="self-center text-[#c0392b] font-black text-2xl">♥</span>
<div class="flex flex-col items-end leading-none rotate-180 scale-75 origin-top-left">
<span class="text-[#c0392b] font-black text-lg">A</span>
<span class="text-[#c0392b] text-xs">♥</span>
</div>
</div>
</div>
</div>
<div class="poker-card-container is-flipped rotate-[5deg] z-10">
<div class="poker-card-inner">
<div class="poker-card-back"></div>
<div class="poker-card-front !p-2 !w-20 !h-28">
<div class="flex flex-col items-start leading-none scale-75 origin-top-left">
<span class="text-[#1a1a2e] font-black text-lg">A</span>
<span class="text-[#1a1a2e] text-xs">♣</span>
</div>
<span class="self-center text-[#1a1a2e] font-black text-2xl">♣</span>
<div class="flex flex-col items-end leading-none rotate-180 scale-75 origin-top-left">
<span class="text-[#1a1a2e] font-black text-lg">A</span>
<span class="text-[#1a1a2e] text-xs">♣</span>
</div>
</div>
</div>
</div>
</div>
</div>
<!-- Other Players -->
<div class="absolute bottom-[10%] left-[-4rem] lg:left-0 flex flex-col items-center z-30">
<div class="w-24 h-24 rounded-full bg-surface-container-low border-2 border-white/5 flex items-center justify-center relative glass-panel">
<span class="text-4xl">👾</span>
<div class="absolute -top-6 -right-12 px-3 py-1 bg-[#145a32] text-[#afcdbd] rounded-lg font-bold text-xs border border-[#d4af37]/20 shadow-lg flex items-center gap-1.5">
<span class="material-symbols-outlined text-[14px] text-[#d4af37]" data-icon="token">token</span>
<span>下注 $500</span>
</div>
<div class="absolute -top-1 -left-1 pos-tag">UTG</div>
</div>
<div class="mt-3 text-center">
<p class="text-on-surface font-bold text-sm tracking-wide">Cypher-7</p>
<p class="text-[#d4af37] font-headline text-xs opacity-80">$22,400</p>
</div>
</div>
<div class="absolute top-[10%] left-[-4rem] lg:left-0 flex flex-col items-center z-30">
<div class="w-24 h-24 rounded-full bg-surface-container-low border-2 border-white/5 flex items-center justify-center relative glass-panel">
<span class="text-4xl opacity-40">🤖</span>
<div class="absolute -bottom-2 -right-8 px-3 py-1 bg-surface-container-highest text-on-surface-variant rounded-lg font-bold text-xs opacity-60 border border-white/5">已弃牌</div>
<div class="absolute -top-1 -right-1 pos-tag">CO</div>
</div>
<div class="mt-3 text-center">
<p class="text-on-surface/40 font-bold text-sm tracking-wide">Unit_99</p>
<p class="text-[#d4af37]/40 font-headline text-xs">$8,900</p>
</div>
</div>
<div class="absolute -top-16 left-1/2 -translate-x-1/2 flex flex-col items-center z-30">
<div class="w-24 h-24 rounded-full bg-surface-container-low border-2 border-white/5 flex items-center justify-center relative glass-panel">
<span class="text-4xl">👽</span>
<div class="absolute -top-1 -right-1 pos-tag animate-tag-glow">BTN</div>
</div>
<div class="mt-3 text-center">
<p class="text-on-surface font-bold text-sm tracking-wide">ZorpX</p>
<p class="text-[#d4af37] font-headline text-xs">$128,400</p>
</div>
</div>
<div class="absolute top-[10%] right-[-4rem] lg:right-0 flex flex-col items-center z-30">
<div class="w-24 h-24 rounded-full bg-surface-container-low border-2 border-white/5 flex items-center justify-center relative glass-panel">
<span class="text-4xl">🎭</span>
<div class="absolute -top-1 -left-1 pos-tag">SB</div>
</div>
<div class="mt-3 text-center">
<p class="text-on-surface font-bold text-sm tracking-wide">Nova-K</p>
<p class="text-[#d4af37] font-headline text-xs">$67,150</p>
</div>
</div>
<div class="absolute bottom-[10%] right-[-4rem] lg:right-0 flex flex-col items-center z-30">
<div class="w-24 h-24 rounded-full bg-surface-container-low border-2 border-white/5 flex items-center justify-center relative glass-panel">
<span class="text-4xl">👩‍🎤</span>
<div class="absolute -top-6 -left-12 px-3 py-1 bg-[#145a32] text-[#afcdbd] rounded-lg font-bold text-xs border border-[#d4af37]/20 shadow-lg flex items-center gap-1.5">
<span class="material-symbols-outlined text-[14px] text-[#d4af37]" data-icon="token">token</span>
<span>下注 $2,500</span>
</div>
<div class="absolute -top-1 -right-1 pos-tag">HJ</div>
</div>
<div class="mt-3 text-center">
<p class="text-on-surface font-bold text-sm tracking-wide">StarDust</p>
<p class="text-[#d4af37] font-headline text-xs">$31,000</p>
</div>
</div>
</div>
</main>
<!-- Bottom Container for HUD and Action Bar -->
<div class="w-full flex flex-col items-center pb-8 z-50 gap-8">
<!-- Hand Strength HUD -->
<div class="w-full max-w-md px-4">
<div class="glass-panel p-5 rounded-2xl border border-[#d4af37]/20 shadow-2xl">
<div class="flex justify-between items-end mb-3">
<span class="text-[#afcdbd] font-bold text-[10px] uppercase tracking-[0.2em]">手牌强度</span>
<span class="text-[#d4af37] font-headline font-black text-xl">三条</span>
</div>
<div class="h-2.5 w-full bg-white/5 rounded-full overflow-hidden">
<div class="h-full w-[65%] strength-bar-gold"></div>
</div>
<div class="flex justify-between mt-2 text-[8px] text-on-surface-variant/60 font-bold uppercase tracking-wider">
<span class="text-[#afcdbd]">高牌</span>
<span class="text-[#afcdbd]">对子</span>
<span class="text-[#d4af37]">三条</span>
<span>葫芦</span>
<span class="opacity-30">同花顺</span>
</div>
</div>
</div>
<!-- Action Bar -->
<footer class="w-full max-w-5xl px-6">
<div class="bg-[#040810]/95 backdrop-blur-2xl rounded-[3rem] border border-[#d4af37]/20 shadow-[0_-15px_50px_rgba(0,0,0,0.8)] py-6 px-8 flex items-center justify-around gap-4">
<button class="flex-1 py-4 px-2 text-[#afcdbd]/80 hover:text-white hover:bg-white/5 rounded-2xl transition-all duration-300 hover:scale-105 font-['Manrope'] font-bold text-xs uppercase flex flex-col items-center gap-2">
<span class="material-symbols-outlined scale-110" data-icon="close">close</span>
<span>弃牌</span>
</button>
<button class="flex-1 py-4 px-2 text-[#afcdbd]/80 hover:text-white hover:bg-white/5 rounded-2xl transition-all duration-300 hover:scale-105 font-['Manrope'] font-bold text-xs uppercase flex flex-col items-center gap-2">
<span class="material-symbols-outlined scale-110" data-icon="done">done</span>
<span>过牌</span>
</button>
<button class="flex-1 py-4 px-2 text-[#afcdbd]/80 hover:text-white hover:bg-white/5 rounded-2xl transition-all duration-300 hover:scale-105 font-['Manrope'] font-bold text-xs uppercase flex flex-col items-center gap-2">
<span class="material-symbols-outlined scale-110" data-icon="payments">payments</span>
<span>跟注 $500</span>
</button>
<!-- Raise Component -->
<div class="flex-[1.8] flex items-center bg-white/5 rounded-2xl p-1.5 gap-2 border border-[#d4af37]/20 transition-all duration-300 hover:shadow-[0_0_25px_rgba(212,175,55,0.2)] hover:border-[#d4af37]/40">
<button class="w-12 h-12 flex items-center justify-center text-[#d4af37] rounded-xl transition-all duration-300 font-black text-2xl raise-btn-hover hover:z-10">-</button>
<button class="flex-1 py-3 text-[#afcdbd] font-['Manrope'] font-bold text-xs uppercase flex flex-col items-center group">
<span class="material-symbols-outlined text-lg group-hover:translate-y-[-2px] transition-transform" data-icon="trending_up">trending_up</span>
<span class="text-[#d4af37] text-sm">加注 $1,250</span>
</button>
<button class="w-12 h-12 flex items-center justify-center text-[#d4af37] rounded-xl transition-all duration-300 font-black text-2xl raise-btn-hover hover:z-10">+</button>
</div>
<button class="flex-1 py-4 px-2 bg-gradient-to-br from-[#d4af37] to-[#b8952c] text-[#060c14] rounded-2xl shadow-[0_0_20px_rgba(212,175,55,0.4)] transition-all duration-300 hover:scale-110 hover:shadow-[0_0_40px_rgba(212,175,55,0.8)] font-['Manrope'] font-black text-xs uppercase flex flex-col items-center gap-2 pulse-gold">
<span class="material-symbols-outlined scale-110 font-bold" data-icon="stars">stars</span>
<span>全押</span>
</button>
</div>
</footer>
</div>
<div class="hidden">
<img data-alt="Dark green premium velvet pool table felt texture" src="https://lh3.googleusercontent.com/aida-public/AB6AXuBkcpMACNWGqZunpRqBiMNw5OgHb849Z6433dx_Ggrrs-P4bc_eEIpZ2Gv6nmLZHvl0ZpHfftIZY37BqedrbZVF1Xzw6R90EiV7flXPesEplGIlnD4R0eytEEnq2xYf0NKjt0BtI2FrRZof6Yu6Z3iiQMA0Ti2dGhuctDMIdFbJa5IfVQBCudNheHhlf-E-ol1ACfYFcfBUTi6-ynqZIXOKTscu1Kk7qldyEtzELwAahuBQ7ODnhVUZGMKtRSZcMNJa6EZmRvOVW-E"/>
<img data-alt="Dark polished mahogany wood grain texture" src="https://lh3.googleusercontent.com/aida-public/AB6AXuAUkTYvzxg6rAEaES4WnjMl1S3w2vL1I0QsUskg3arlw84kC0TOv739yV98PuhfPo6W6mLL9TsuxHsWWcGa8N4bjA5RjE2Y81Qrft4Rfoc749EtNw9O2q6x4b6ddNEiJjCdPyrJZd71YPIIBjBJVhdtzSIW_mqoswY89GZRSwYhjLGLjRDCaAm2aqwJzgW8T97Jw2PjS4FzCRtFsq3NMaYgr7hPrGVXPHVJdacIxwrp5fS8wQWoa7R-NJf4MXuhosyxugtk3eTUnj4"/>
</div>
</body></html>
~~~
