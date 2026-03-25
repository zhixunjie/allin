-- 测试账号种子数据（密码均为 123456）
-- 使用 INSERT IGNORE 保证重复执行时幂等

USE allin;

INSERT IGNORE INTO users (username, password_hash, display_name, chip_balance) VALUES
  ('test1', '$2a$10$MBhQerFWRKXL7c0JTBmvm.VJtgi6OUWTPWsDe4pXgA5FiTSssN4M6', '测试玩家1', 10000),
  ('test2', '$2a$10$7b1x0bz0wNSz.20ypPceVeSBlMulUloAp8Swfdusalqqg12e66.sG',  '测试玩家2', 10000),
  ('test3', '$2a$10$SysMS1Eu6HZaehjOQqb82urNg2vH8wJg7xU2oCdGmPIy1g5/bgiW.', '测试玩家3', 10000),
  ('test4', '$2a$10$d5h9S8WHIGTN7iw1R4a9HOeQUidGlZNXTsDbP3HOO7dqjkwiegAjy', '测试玩家4', 10000);
