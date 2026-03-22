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
  
- 项目说明：[PROJECT.md](PROJECT.md)
  
- 开发进度：[PROGRESS.md](PROGRESS.md)
  
- 视觉设计风格：[DESIGN.md](DESIGN.md) https://stitch.withgoogle.com/projects/4278287175556993233?pli=1
  
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

