-- ============================================================
-- 既存ローカルDBを今回の追加機能に対応させるための差分SQL
-- すでに001_init.sqlを実行済みの場合はこちらを追加で実行してください。
-- 新しくDBを作り直す場合は001_init.sqlだけで十分です。
-- ============================================================

-- status に canceled を追加します。
ALTER TABLE items
  MODIFY status ENUM('available', 'sold', 'canceled') NOT NULL DEFAULT 'available';

-- コメント返信と最終更新日時を扱うための列を追加します。
ALTER TABLE messages
  ADD COLUMN parent_message_id BIGINT NULL AFTER item_id,
  ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at,
  ADD INDEX idx_messages_parent_message_id (parent_message_id),
  ADD INDEX idx_messages_item_updated_at (item_id, updated_at),
  ADD CONSTRAINT fk_messages_parent FOREIGN KEY (parent_message_id) REFERENCES messages(id) ON DELETE CASCADE;

-- 気になる商品を保存するチェックリスト。
CREATE TABLE IF NOT EXISTS checklist (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  item_id BIGINT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_checklist_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_checklist_item FOREIGN KEY (item_id) REFERENCES items(id),
  UNIQUE KEY uq_checklist_user_item (user_id, item_id),
  INDEX idx_checklist_user_created_at (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
