package repository

import (
	"context"
	"database/sql"

	"hackathon-backend/internal/models"
)

// MessageRepository は messages テーブルへのDB操作を担当します。
type MessageRepository struct {
	DB *sql.DB
}

// ListByItem は商品IDに紐づくメッセージ一覧を作成日時順に取得します。
func (r *MessageRepository) ListByItem(ctx context.Context, itemID int64) ([]models.Message, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT m.id, m.item_id, m.sender_id, su.name, m.receiver_id, ru.name, m.body, m.created_at
         FROM messages m
         JOIN users su ON su.id = m.sender_id
         JOIN users ru ON ru.id = m.receiver_id
         WHERE m.item_id = ?
         ORDER BY m.created_at ASC`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ID,
			&msg.ItemID,
			&msg.SenderID,
			&msg.SenderName,
			&msg.ReceiverID,
			&msg.ReceiverName,
			&msg.Body,
			&msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Create は新しいDMを保存します。
func (r *MessageRepository) Create(ctx context.Context, itemID, senderID, receiverID int64, body string) (models.Message, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO messages (item_id, sender_id, receiver_id, body) VALUES (?, ?, ?, ?)`,
		itemID,
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

	var msg models.Message
	err = r.DB.QueryRowContext(
		ctx,
		`SELECT m.id, m.item_id, m.sender_id, su.name, m.receiver_id, ru.name, m.body, m.created_at
         FROM messages m
         JOIN users su ON su.id = m.sender_id
         JOIN users ru ON ru.id = m.receiver_id
         WHERE m.id = ?`,
		id,
	).Scan(
		&msg.ID,
		&msg.ItemID,
		&msg.SenderID,
		&msg.SenderName,
		&msg.ReceiverID,
		&msg.ReceiverName,
		&msg.Body,
		&msg.CreatedAt,
	)
	if err != nil {
		return models.Message{}, err
	}
	return msg, nil
}
