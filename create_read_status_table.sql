-- 创建 message_read_status 表
-- 在 MySQL 中执行此脚本

USE chatdb;

-- 创建消息已读状态表
CREATE TABLE IF NOT EXISTS `message_read_status` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `message_id` bigint unsigned NOT NULL COMMENT '消息ID',
  `reader_id` bigint unsigned NOT NULL COMMENT '读取者用户ID',
  `read_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '读取时间',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_message_reader` (`message_id`,`reader_id`) COMMENT '防止重复标记已读',
  KEY `idx_read_status_message` (`message_id`) COMMENT '按消息查询已读记录',
  KEY `idx_read_status_reader` (`reader_id`) COMMENT '按用户查询已读消息',
  KEY `idx_read_status_time` (`read_at`) COMMENT '按时间查询'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='消息已读状态表';

-- 验证表已创建
SELECT 'message_read_status 表创建成功！' AS status;

-- 显示表结构
DESC message_read_status;

-- 显示所有表
SHOW TABLES;
