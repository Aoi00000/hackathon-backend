package repository

import (
	"context"
	"database/sql"
	"fmt"

	"hackathon-backend/internal/models"
)

// MessageRepository は公開コメントと非公開DMへのDB操作を担当します。
type MessageRepository struct{ DB *sql.DB }

func (r *MessageRepository) ListByItem(ctx context.Context, itemID int64) ([]models.Message, error) {
	rows, err := r.DB.QueryContext(ctx,
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
                  m.created_at ASC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []models.Message{}
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func scanMessage(scanner interface{ Scan(dest ...any) error }) (models.Message, error) {
	var msg models.Message
	var parentID sql.NullInt64
	var isSeller int
	err := scanner.Scan(&msg.ID, &msg.ItemID, &parentID, &msg.SenderID, &msg.SenderName, &msg.ReceiverID, &msg.ReceiverName, &msg.Body, &isSeller, &msg.CreatedAt, &msg.UpdatedAt)
	if parentID.Valid {
		v := parentID.Int64
		msg.ParentMessageID = &v
	}
	msg.IsSeller = isSeller == 1
	return msg, err
}

func (r *MessageRepository) Create(ctx context.Context, itemID, senderID int64, parentMessageID *int64, body string) (models.Message, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Message{}, err
	}
	defer tx.Rollback()
	var receiverID, sellerID int64
	if err := tx.QueryRowContext(ctx, `SELECT seller_id FROM items WHERE id=?`, itemID).Scan(&sellerID); err != nil {
		return models.Message{}, err
	}
	var blocked int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM blocked_users WHERE (blocker_id=? AND blocked_id=?) OR (blocker_id=? AND blocked_id=?)`, senderID, sellerID, sellerID, senderID).Scan(&blocked); err != nil {
		return models.Message{}, err
	}
	if blocked > 0 {
		return models.Message{}, fmt.Errorf("ブロック関係にあるためコメントできません")
	}
	if parentMessageID != nil {
		var parentItemID int64
		if err := tx.QueryRowContext(ctx, `SELECT item_id, sender_id FROM messages WHERE id=?`, *parentMessageID).Scan(&parentItemID, &receiverID); err != nil {
			return models.Message{}, err
		}
		if parentItemID != itemID {
			return models.Message{}, fmt.Errorf("返信先コメントが商品と一致しません")
		}
	} else {
		receiverID = sellerID
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO messages (item_id,parent_message_id,sender_id,receiver_id,body) VALUES (?,?,?,?,?)`, itemID, parentMessageID, senderID, receiverID, body)
	if err != nil {
		return models.Message{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Message{}, err
	}
	if parentMessageID != nil {
		_, _ = tx.ExecContext(ctx, `UPDATE messages SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, *parentMessageID)
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id,item_id,title,body) VALUES (?, ?, 'コメント通知', '商品にコメントまたは返信が追加されました')`, receiverID, itemID)
	if err := tx.Commit(); err != nil {
		return models.Message{}, err
	}
	return r.FindByID(ctx, id)
}

func (r *MessageRepository) FindByID(ctx context.Context, id int64) (models.Message, error) {
	return scanMessage(r.DB.QueryRowContext(ctx,
		`SELECT m.id, m.item_id, m.parent_message_id, m.sender_id, su.name, m.receiver_id, ru.name,
                m.body, CASE WHEN m.sender_id = i.seller_id THEN 1 ELSE 0 END AS is_seller, m.created_at, m.updated_at
         FROM messages m JOIN items i ON i.id=m.item_id JOIN users su ON su.id=m.sender_id JOIN users ru ON ru.id=m.receiver_id WHERE m.id=?`, id))
}

func (r *MessageRepository) ListPrivateByItem(ctx context.Context, itemID, userID int64) ([]models.PrivateMessage, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT p.id, p.item_id, p.parent_private_message_id, p.sender_id, su.name, p.receiver_id, ru.name, p.body, p.created_at
         FROM private_messages p
         JOIN users su ON su.id = p.sender_id
         JOIN users ru ON ru.id = p.receiver_id
         LEFT JOIN private_messages parent ON parent.id = p.parent_private_message_id
         WHERE p.item_id = ? AND (p.sender_id = ? OR p.receiver_id = ?)
         ORDER BY COALESCE(parent.created_at, p.created_at) DESC,
                  CASE WHEN p.parent_private_message_id IS NULL THEN 0 ELSE 1 END ASC,
                  p.created_at ASC`, itemID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PrivateMessage
	for rows.Next() {
		var m models.PrivateMessage
		var parentID sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ItemID, &parentID, &m.SenderID, &m.SenderName, &m.ReceiverID, &m.ReceiverName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			m.ParentPrivateMessageID = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MessageRepository) CreatePrivate(ctx context.Context, itemID, senderID, receiverID int64, parentMessageID *int64, body string) (models.PrivateMessage, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.PrivateMessage{}, err
	}
	defer tx.Rollback()
	var sellerID int64
	if err := tx.QueryRowContext(ctx, `SELECT seller_id FROM items WHERE id=?`, itemID).Scan(&sellerID); err != nil {
		return models.PrivateMessage{}, err
	}
	if parentMessageID != nil {
		var parentItemID, parentSenderID, parentReceiverID int64
		if err := tx.QueryRowContext(ctx, `SELECT item_id, sender_id, receiver_id FROM private_messages WHERE id=?`, *parentMessageID).Scan(&parentItemID, &parentSenderID, &parentReceiverID); err != nil {
			return models.PrivateMessage{}, err
		}
		if parentItemID != itemID {
			return models.PrivateMessage{}, fmt.Errorf("返信先DMが商品と一致しません")
		}
		if senderID != parentSenderID && senderID != parentReceiverID {
			return models.PrivateMessage{}, fmt.Errorf("このDMスレッドには返信できません")
		}
		if receiverID == 0 {
			if senderID == parentSenderID {
				receiverID = parentReceiverID
			} else {
				receiverID = parentSenderID
			}
		}
	}
	if receiverID == 0 {
		receiverID = sellerID
	}
	if senderID != sellerID && receiverID != sellerID {
		return models.PrivateMessage{}, fmt.Errorf("DMは購入検討者と出品者の間でのみ送れます")
	}
	var blocked int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM blocked_users WHERE (blocker_id=? AND blocked_id=?) OR (blocker_id=? AND blocked_id=?)`, senderID, receiverID, receiverID, senderID).Scan(&blocked); err != nil {
		return models.PrivateMessage{}, err
	}
	if blocked > 0 {
		return models.PrivateMessage{}, fmt.Errorf("ブロック関係にあるためDMできません")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO private_messages (item_id,parent_private_message_id,sender_id,receiver_id,body) VALUES (?,?,?,?,?)`, itemID, parentMessageID, senderID, receiverID, body)
	if err != nil {
		return models.PrivateMessage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.PrivateMessage{}, err
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id,item_id,title,body) VALUES (?, ?, 'DM通知', '商品について非公開DMが届きました')`, receiverID, itemID)
	if err := tx.Commit(); err != nil {
		return models.PrivateMessage{}, err
	}
	return r.FindPrivateByID(ctx, id)
}

func (r *MessageRepository) FindPrivateByID(ctx context.Context, id int64) (models.PrivateMessage, error) {
	var m models.PrivateMessage
	var parentID sql.NullInt64
	err := r.DB.QueryRowContext(ctx, `SELECT p.id,p.item_id,p.parent_private_message_id,p.sender_id,su.name,p.receiver_id,ru.name,p.body,p.created_at FROM private_messages p JOIN users su ON su.id=p.sender_id JOIN users ru ON ru.id=p.receiver_id WHERE p.id=?`, id).Scan(&m.ID, &m.ItemID, &parentID, &m.SenderID, &m.SenderName, &m.ReceiverID, &m.ReceiverName, &m.Body, &m.CreatedAt)
	if parentID.Valid {
		v := parentID.Int64
		m.ParentPrivateMessageID = &v
	}
	return m, err
}

func (r *MessageRepository) DeletePublicBySeller(ctx context.Context, itemID, messageID, sellerID int64) error {
	// 出品者だけが自分の商品についた公開コメントを削除できます。
	// 親コメントを削除した場合、DBのON DELETE CASCADEにより返信もまとめて削除されます。
	result, err := r.DB.ExecContext(ctx, `
		DELETE m FROM messages m
		JOIN items i ON i.id = m.item_id
		WHERE m.id = ? AND m.item_id = ? AND i.seller_id = ?`, messageID, itemID, sellerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("削除できるコメントが見つかりません")
	}
	return nil
}
