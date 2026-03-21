-- AllIn 数据库 DDL
-- 数据库: allin
-- 更新时间: 2026-03-22

CREATE DATABASE IF NOT EXISTS allin
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE allin;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '用户自增ID',
    username      VARCHAR(32)  NOT NULL                COMMENT '登录用户名',
    password_hash VARCHAR(60)  NOT NULL                COMMENT 'bcrypt 密码哈希',
    display_name  VARCHAR(32)  NOT NULL                COMMENT '显示昵称',
    chip_balance  BIGINT       NOT NULL DEFAULT 10000  COMMENT '账户筹码余额',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
    UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账户表';

-- 房间历史表
CREATE TABLE IF NOT EXISTS room_history (
    id           BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '房间记录自增ID',
    room_code    CHAR(6)     NOT NULL             COMMENT '6位房间码',
    host_user_id BIGINT      NOT NULL             COMMENT '房主用户ID',
    config_json  JSON        NOT NULL             COMMENT '房间配置快照（RoomConfig JSON）',
    started_at   DATETIME    NOT NULL             COMMENT '房间创建时间',
    ended_at     DATETIME                         COMMENT '房间关闭时间；NULL 表示仍在进行',
    INDEX idx_room_code (room_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='房间生命周期历史表';

-- 手牌历史表
CREATE TABLE IF NOT EXISTS hand_history (
    id            BIGINT    NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '手牌记录自增ID',
    room_id       BIGINT    NOT NULL             COMMENT '所属房间ID',
    hand_num      INT       NOT NULL             COMMENT '房间内手牌序号（从1开始）',
    players_json  JSON      NOT NULL             COMMENT '参与玩家快照（座位/筹码等）',
    actions_json  JSON      NOT NULL             COMMENT '本手牌所有行动序列',
    result_json   JSON      NOT NULL             COMMENT '手牌结果（赢家/金额）',
    played_at     DATETIME  NOT NULL             COMMENT '手牌结束时间',
    INDEX idx_hand_room (room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单手牌历史记录表';

-- 筹码流水表
CREATE TABLE IF NOT EXISTS chip_ledger (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '流水自增ID',
    user_id     BIGINT       NOT NULL                           COMMENT '用户ID',
    delta       BIGINT       NOT NULL                           COMMENT '筹码变动量（正=收入，负=支出）',
    reason      VARCHAR(32)  NOT NULL                           COMMENT '变动原因（buy_in/cash_out/add_chips）',
    ref_id      VARCHAR(32)                                     COMMENT '关联业务ID（如房间码）',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录时间',
    INDEX idx_ledger_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='筹码变动流水表';
