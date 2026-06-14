-- ============================================================
-- ハッカソン用フリマアプリ 初期DBスキーマ
-- MySQL 8.0 / utf8mb4 を想定
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(80) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  balance_coins INT NOT NULL DEFAULT 0,
  sales_coins INT NOT NULL DEFAULT 0,
  rating_sum INT NOT NULL DEFAULT 0,
  rating_count INT NOT NULL DEFAULT 0,
  transaction_count INT NOT NULL DEFAULT 0,
  shipping_region VARCHAR(80) NULL,
  shipping_address TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS items (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  product_code VARCHAR(32) UNIQUE,
  seller_id BIGINT NOT NULL,
  title VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  category VARCHAR(80) NOT NULL,
  condition_text VARCHAR(80) NOT NULL,
  price_yen INT NOT NULL,
  image_url MEDIUMTEXT NULL,
  delivery_method VARCHAR(120) NOT NULL DEFAULT '対面・配送相談',
  shipping_days INT NOT NULL DEFAULT 2,
  ship_from_region VARCHAR(80) NOT NULL DEFAULT '未設定',
  size VARCHAR(80) NULL,
  color VARCHAR(80) NULL,
  tags TEXT NULL,
  status ENUM('available', 'sold', 'canceled') NOT NULL DEFAULT 'available',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_items_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  INDEX idx_items_status_updated_at (status, updated_at),
  INDEX idx_items_seller_id (seller_id),
  INDEX idx_items_category_status (category, status),
  FULLTEXT INDEX ft_items_title_description (title, description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS purchases (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL UNIQUE,
  buyer_id BIGINT NOT NULL,
  seller_id BIGINT NOT NULL,
  price_yen INT NOT NULL,
  status ENUM('paid', 'shipped', 'completed', 'canceled') NOT NULL DEFAULT 'paid',
  delivery_address TEXT NULL,
  shipping_deadline DATETIME NOT NULL,
  shipped_at DATETIME NULL,
  completed_at DATETIME NULL,
  rating INT NULL,
  rating_comment TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_purchases_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_purchases_buyer FOREIGN KEY (buyer_id) REFERENCES users(id),
  CONSTRAINT fk_purchases_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  INDEX idx_purchases_buyer_created_at (buyer_id, created_at),
  INDEX idx_purchases_seller_id (seller_id),
  INDEX idx_purchases_status_deadline (status, shipping_deadline)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL,
  parent_message_id BIGINT NULL,
  sender_id BIGINT NOT NULL,
  receiver_id BIGINT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_messages_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_messages_parent FOREIGN KEY (parent_message_id) REFERENCES messages(id) ON DELETE CASCADE,
  CONSTRAINT fk_messages_sender FOREIGN KEY (sender_id) REFERENCES users(id),
  CONSTRAINT fk_messages_receiver FOREIGN KEY (receiver_id) REFERENCES users(id),
  INDEX idx_messages_item_updated_at (item_id, updated_at),
  INDEX idx_messages_parent_message_id (parent_message_id),
  INDEX idx_messages_receiver_id (receiver_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS private_messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL,
  parent_private_message_id BIGINT NULL,
  sender_id BIGINT NOT NULL,
  receiver_id BIGINT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_private_messages_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_private_messages_parent FOREIGN KEY (parent_private_message_id) REFERENCES private_messages(id) ON DELETE CASCADE,
  CONSTRAINT fk_private_messages_sender FOREIGN KEY (sender_id) REFERENCES users(id),
  CONSTRAINT fk_private_messages_receiver FOREIGN KEY (receiver_id) REFERENCES users(id),
  INDEX idx_private_messages_item_users (item_id, sender_id, receiver_id),
  INDEX idx_private_messages_parent (parent_private_message_id),
  INDEX idx_private_messages_receiver_created_at (receiver_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS checklist (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  item_id BIGINT NOT NULL,
  last_seen_updated_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_checklist_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_checklist_item FOREIGN KEY (item_id) REFERENCES items(id),
  UNIQUE KEY uq_checklist_user_item (user_id, item_id),
  INDEX idx_checklist_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
  subject VARCHAR(120) NOT NULL DEFAULT '一般相談',
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_support_messages_user FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_support_messages_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
