package repository

import (
	"context"
	"database/sql"
	"fmt"

	"hackathon-backend/internal/models"
)

// MessageRepository は messages テーブルへのDB操作を担当します。
// 旧来の1対1 DMではなく、商品ページに紐づくコメント欄として利用します。
type MessageRepository struct {
	DB *sql.DB
}

// ListByItem は商品IDに紐づくコメント一覧を取得します。
// 親コメントは「最後に返信された時刻」をupdated_atとして更新するため、最新の議論が上に来ます。
func (r *MessageRepository) ListByItem(ctx context.Context, itemID int64) ([]models.Message, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT m.id, m.item_id, m.parent_message_id, m.sender_id, su.name, m.receiver_id, ru.name,
                m.body, CASE WHEN m.sender_id = i.seller_id THEN 1 ELSE 0 END AS is_seller, m.created_at, m.updated_at
         FROM messages m
         JOIN items i ON i.id = m.item_id
         JOIN users su ON su.id = m.sender_id
         JOIN users ru ON ru.id = m.receiver_id
         LEFT JOIN messages parent ON parent.id = m.parent_message_id
         WHERE m.item_id = ?
         ORDER BY COALESCE(parent.updated_at, m.updated_at) DESC,
                  CASE WHEN m.parent_message_id IS NULL THEN 0 ELSE 1 END ASC,
                  m.created_at ASC`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		var parentID sql.NullInt64
		var isSellerInt int
		if err := rows.Scan(
			&msg.ID,
			&msg.ItemID,
			&parentID,
			&msg.SenderID,
			&msg.SenderName,
			&msg.ReceiverID,
			&msg.ReceiverName,
			&msg.Body,
			&isSellerInt,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			msg.ParentMessageID = &v
		}
		msg.IsSeller = isSellerInt == 1
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Create は新しいコメントまたは返信を保存します。
// receiver_idは、親コメントなら出品者、返信なら返信先コメントの投稿者に自動設定します。
func (r *MessageRepository) Create(ctx context.Context, itemID, senderID int64, parentMessageID *int64, body string) (models.Message, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Message{}, err
	}
	defer tx.Rollback()

	var receiverID int64
	var sellerID int64
	if err := tx.QueryRowContext(ctx, `SELECT seller_id FROM items WHERE id = ?`, itemID).Scan(&sellerID); err != nil {
		return models.Message{}, err
	}

	if parentMessageID != nil {
		var parentItemID int64
		if err := tx.QueryRowContext(
			ctx,
			`SELECT item_id, sender_id FROM messages WHERE id = ?`,
			*parentMessageID,
		).Scan(&parentItemID, &receiverID); err != nil {
			return models.Message{}, err
		}
		if parentItemID != itemID {
			return models.Message{}, fmt.Errorf("返信先コメントが商品と一致しません")
		}
	} else {
		receiverID = sellerID
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO messages (item_id, parent_message_id, sender_id, receiver_id, body)
         VALUES (?, ?, ?, ?, ?)`,
		itemID,
		parentMessageID,
		senderID,
		receiverID,
		body,
	)
	if err != nil {
		return models.Message{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Message{}, err
	}

	// 返信が追加されたときに親コメントのupdated_atを更新し、スレッド全体を上に上げます。
	if parentMessageID != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE messages SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *parentMessageID); err != nil {
			return models.Message{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Message{}, err
	}

	return r.FindByID(ctx, id)
}

// FindByID は作成直後のコメントをレスポンスとして返すために利用します。
func (r *MessageRepository) FindByID(ctx context.Context, id int64) (models.Message, error) {
	var msg models.Message
	var parentID sql.NullInt64
	var isSellerInt int
	err := r.DB.QueryRowContext(
		ctx,
		`SELECT m.id, m.item_id, m.parent_message_id, m.sender_id, su.name, m.receiver_id, ru.name,
                m.body, CASE WHEN m.sender_id = i.seller_id THEN 1 ELSE 0 END AS is_seller, m.created_at, m.updated_at
         FROM messages m
         JOIN items i ON i.id = m.item_id
         JOIN users su ON su.id = m.sender_id
         JOIN users ru ON ru.id = m.receiver_id
         WHERE m.id = ?`,
		id,
	).Scan(
		&msg.ID,
		&msg.ItemID,
		&parentID,
		&msg.SenderID,
		&msg.SenderName,
		&msg.ReceiverID,
		&msg.ReceiverName,
		&msg.Body,
		&isSellerInt,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)
	if err != nil {
		return models.Message{}, err
	}
	if parentID.Valid {
		v := parentID.Int64
		msg.ParentMessageID = &v
	}
	msg.IsSeller = isSellerInt == 1
	return msg, nil
}
