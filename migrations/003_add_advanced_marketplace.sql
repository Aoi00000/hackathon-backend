-- ============================================================
-- 既存DBを高度な購入フロー、残高、通知、DM、検索保存、ブロックに対応させる差分SQL
-- 001_init.sql + 002_add_history_comments_checklist.sql 実行済みのDBで1回だけ実行してください。
-- ============================================================

ALTER TABLE users
  ADD COLUMN balance_coins INT NOT NULL DEFAULT 0 AFTER password_hash,
  ADD COLUMN sales_coins INT NOT NULL DEFAULT 0 AFTER balance_coins,
  ADD COLUMN rating_sum INT NOT NULL DEFAULT 0 AFTER sales_coins,
  ADD COLUMN rating_count INT NOT NULL DEFAULT 0 AFTER rating_sum,
  ADD COLUMN transaction_count INT NOT NULL DEFAULT 0 AFTER rating_count,
  ADD COLUMN shipping_region VARCHAR(80) NULL AFTER transaction_count,
  ADD COLUMN shipping_address TEXT NULL AFTER shipping_region;

ALTER TABLE items
  ADD COLUMN product_code VARCHAR(32) UNIQUE AFTER id,
  ADD COLUMN delivery_method VARCHAR(120) NOT NULL DEFAULT '対面・配送相談' AFTER image_url,
  ADD COLUMN shipping_days INT NOT NULL DEFAULT 2 AFTER delivery_method,
  ADD COLUMN ship_from_region VARCHAR(80) NOT NULL DEFAULT '未設定' AFTER shipping_days,
  ADD COLUMN size VARCHAR(80) NULL AFTER ship_from_region,
  ADD COLUMN color VARCHAR(80) NULL AFTER size,
  ADD COLUMN tags TEXT NULL AFTER color;

UPDATE items SET product_code = CONCAT('AFM-', LPAD(id, 6, '0')) WHERE product_code IS NULL;

ALTER TABLE purchases
  ADD COLUMN status ENUM('paid', 'shipped', 'completed', 'canceled') NOT NULL DEFAULT 'paid' AFTER price_yen,
  ADD COLUMN delivery_address TEXT NULL AFTER status,
  ADD COLUMN shipping_deadline DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP AFTER delivery_address,
  ADD COLUMN shipped_at DATETIME NULL AFTER shipping_deadline,
  ADD COLUMN completed_at DATETIME NULL AFTER shipped_at,
  ADD COLUMN rating INT NULL AFTER completed_at,
  ADD COLUMN rating_comment TEXT NULL AFTER rating;

CREATE TABLE IF NOT EXISTS private_messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL,
  sender_id BIGINT NOT NULL,
  receiver_id BIGINT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_private_messages_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_private_messages_sender FOREIGN KEY (sender_id) REFERENCES users(id),
  CONSTRAINT fk_private_messages_receiver FOREIGN KEY (receiver_id) REFERENCES users(id),
  INDEX idx_private_messages_item_users (item_id, sender_id, receiver_id),
  INDEX idx_private_messages_receiver_created_at (receiver_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE checklist
  ADD COLUMN last_seen_updated_at DATETIME NULL AFTER item_id;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  item_id BIGINT NULL,
  title VARCHAR(120) NOT NULL,
  body TEXT NOT NULL,
  read_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_notifications_item FOREIGN KEY (item_id) REFERENCES items(id),
  INDEX idx_notifications_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS saved_searches (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  name VARCHAR(120) NOT NULL,
  query_json TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_saved_searches_user FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_saved_searches_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blocked_users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  blocker_id BIGINT NOT NULL,
  blocked_id BIGINT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_blocked_users_blocker FOREIGN KEY (blocker_id) REFERENCES users(id),
  CONSTRAINT fk_blocked_users_blocked FOREIGN KEY (blocked_id) REFERENCES users(id),
  UNIQUE KEY uq_blocked_users_pair (blocker_id, blocked_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS support_messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_support_messages_user FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_support_messages_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
