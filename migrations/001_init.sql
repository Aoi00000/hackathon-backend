-- ============================================================
-- ハッカソン用フリマアプリ 初期DBスキーマ
-- MySQL 8.0 / utf8mb4 を想定
-- ============================================================

-- ユーザー情報。
-- パスワードそのものは保存せず、bcryptでハッシュ化した値だけを保存する。
CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(80) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 商品情報。
-- 画像アップロード機能まで作ると時間がかかるため、MVPでは画像URLを保存する設計にしている。
CREATE TABLE IF NOT EXISTS items (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  seller_id BIGINT NOT NULL,
  title VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  category VARCHAR(80) NOT NULL,
  condition_text VARCHAR(80) NOT NULL,
  price_yen INT NOT NULL,
  image_url TEXT NULL,
  status ENUM('available', 'sold') NOT NULL DEFAULT 'available',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_items_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  INDEX idx_items_status_created_at (status, created_at),
  INDEX idx_items_seller_id (seller_id),
  FULLTEXT INDEX ft_items_title_description (title, description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 購入情報。
-- item_id に UNIQUE を付けることで、同じ商品が二重購入される事故をDB側でも防ぐ。
CREATE TABLE IF NOT EXISTS purchases (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL UNIQUE,
  buyer_id BIGINT NOT NULL,
  seller_id BIGINT NOT NULL,
  price_yen INT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_purchases_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_purchases_buyer FOREIGN KEY (buyer_id) REFERENCES users(id),
  CONSTRAINT fk_purchases_seller FOREIGN KEY (seller_id) REFERENCES users(id),
  INDEX idx_purchases_buyer_id (buyer_id),
  INDEX idx_purchases_seller_id (seller_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 商品に紐づくDM。
-- item_idを持たせることで「この商品についての会話」を表現しやすくする。
CREATE TABLE IF NOT EXISTS messages (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id BIGINT NOT NULL,
  sender_id BIGINT NOT NULL,
  receiver_id BIGINT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_messages_item FOREIGN KEY (item_id) REFERENCES items(id),
  CONSTRAINT fk_messages_sender FOREIGN KEY (sender_id) REFERENCES users(id),
  CONSTRAINT fk_messages_receiver FOREIGN KEY (receiver_id) REFERENCES users(id),
  INDEX idx_messages_item_created_at (item_id, created_at),
  INDEX idx_messages_receiver_id (receiver_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
