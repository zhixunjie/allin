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

# TMP

~~~
                                                                                 
⏺ 基于已读的代码，PixiJS 负责整个游戏牌桌的 2D 渲染，全部程序化绘制（无外部图片）：
                                                                                   
  场景层 (TableScene.ts)                                                           
                                                                                   
  - 太空背景 — 深色底 + 120颗伪随机星星 + 金色环境光晕                             
  - 牌桌 — 木质边框椭圆 → 金色描边 → 多层绿色毛毡（模拟径向渐变）→ 内圈金色装饰线
  - 公共牌 — 5张牌位，有牌显示正面，未发的显示背面，idle时隐藏                     
  - 底池显示 — 桌面中央下方                                                        
  - 庄家按钮 — 金色圆形 "D" 标记                                                   
  - 街道标签 — PREFLOP / FLOP / TURN / RIVER                                       
  - 筹码飞行动画 — 街道切换时筹码从各座位飞向底池                                  
  - 发牌动画 — DealAnimation                                                       
                                                                                   
  座位层 (SeatSprite.ts)                                                           
                                                                                   
  - 9个座位 — 本地玩家固定在底部（display index 0），其他玩家相对旋转              
  - 头像 — 圆形玻璃面板，emoji 图标（根据 userId hash），机器人显示🤖           
  - 位置标签 — BTN / SB / BB / UTG 等，金色胶囊标签                                
  - 名字 — 本地玩家用金色徽章，远端玩家普通文字                                    
  - 筹码面板 — 本地玩家有专属大号筹码余额面板                                      
  - 下注徽章 — 朝桌心方向偏移的绿色胶囊                                            
  - 状态徽章 — "已弃牌" / "ALL-IN"                                                 
  - 手牌 — 本地玩家在头像右侧扇形展开，远端玩家在头像上方（showdown时）            
  - 行动高亮 — 当前行动玩家头像金色光环                                            
  - 弃牌变暗 — 整体 alpha 降低                                                     
                                                                                   
  卡牌层 (CardSprite.ts)                                                           
                                                                                   
  - 正面 — 奶白底 + 金色边框 + 左上角花色/点数 + 中央大花色符号                    
  - 背面 — 深绿底 + 菱形交叉线纹理 + 中央径向光晕 + ♦装饰 + 金色边框
  - 两种尺寸：公共牌（80×112）和手牌（64×90）                                      
                                                                                   
  计时器 (TimerArc.ts)                                                             
                                                                                   
  - 环绕当前行动玩家头像的弧形倒计时，跟随 deadline 实时更新                       
   
  动画                                                                             
                                                                                
  - ChipAnimation — 筹码飞向底池                                                   
  - DealAnimation — 发牌动画                                                    
                                                                                   
  总结：PixiJS 承担了所有游戏视觉元素，React 只负责操作按钮、聊天、连接状态等 UI   
  控件，两者通过 Zustand store 同步状态。                                          
                                           
~~~

