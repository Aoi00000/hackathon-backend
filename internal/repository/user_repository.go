package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"hackathon-backend/internal/models"
)

// UserRepository は users とユーザー周辺テーブルへのDB操作を担当します。
type UserRepository struct {
	DB *sql.DB
}

func (r *UserRepository) Create(ctx context.Context, name, email, passwordHash string) (models.User, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO users (name, email, password_hash, balance_coins, sales_coins) VALUES (?, ?, ?, 0, 0)`,
		name,
		email,
		passwordHash,
	)
	if err != nil {
		return models.User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}
	return r.FindByID(ctx, id)
}

func scanUser(scanner interface{ Scan(dest ...any) error }) (models.User, error) {
	var user models.User
	var ratingAverage sql.NullFloat64
	var shippingRegion sql.NullString
	var shippingAddress sql.NullString
	err := scanner.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.BalanceCoins,
		&user.SalesCoins,
		&ratingAverage,
		&user.RatingCount,
		&user.TransactionCount,
		&shippingRegion,
		&shippingAddress,
		&user.CreatedAt,
	)
	if ratingAverage.Valid {
		user.RatingAverage = ratingAverage.Float64
	}
	if shippingRegion.Valid {
		user.ShippingRegion = shippingRegion.String
	}
	if shippingAddress.Valid {
		user.ShippingAddress = shippingAddress.String
	}
	return user, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	user, err := scanUser(r.DB.QueryRowContext(
		ctx,
		`SELECT id, name, email, password_hash, balance_coins, sales_coins,
                CASE WHEN rating_count = 0 THEN 0 ELSE rating_sum / rating_count END AS rating_average,
                rating_count, transaction_count, shipping_region, shipping_address, created_at
         FROM users WHERE email = ?`,
		email,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, err
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (models.User, error) {
	user, err := scanUser(r.DB.QueryRowContext(
		ctx,
		`SELECT id, name, email, password_hash, balance_coins, sales_coins,
                CASE WHEN rating_count = 0 THEN 0 ELSE rating_sum / rating_count END AS rating_average,
                rating_count, transaction_count, shipping_region, shipping_address, created_at
         FROM users WHERE id = ?`,
		id,
	))
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

// Charge はアプリ内仮想通貨をチャージします。
func (r *UserRepository) Charge(ctx context.Context, userID int64, amount int) (models.User, error) {
	if amount <= 0 {
		return models.User{}, fmt.Errorf("チャージ金額は1以上にしてください")
	}
	if amount > 1000000 {
		return models.User{}, fmt.Errorf("一度にチャージできる上限を超えています")
	}
	if _, err := r.DB.ExecContext(ctx, `UPDATE users SET balance_coins = balance_coins + ? WHERE id = ?`, amount, userID); err != nil {
		return models.User{}, err
	}
	// チャージはユーザーにとって重要な残高変動なので、通知一覧にも記録します。
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, 'チャージ完了', ?)`, userID, fmt.Sprintf("%dコインをチャージしました", amount))
	return r.FindByID(ctx, userID)
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID int64, req models.UpdateProfileRequest) (models.User, error) {
	_, err := r.DB.ExecContext(
		ctx,
		`UPDATE users SET shipping_region = ?, shipping_address = ? WHERE id = ?`,
		req.ShippingRegion,
		req.ShippingAddress,
		userID,
	)
	if err != nil {
		return models.User{}, err
	}
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, '住所保存完了', '発送元・お届け先住所を保存しました')`, userID)
	return r.FindByID(ctx, userID)
}

func (r *UserRepository) BlockUser(ctx context.Context, blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return fmt.Errorf("自分自身はブロックできません")
	}
	_, err := r.DB.ExecContext(ctx, `INSERT IGNORE INTO blocked_users (blocker_id, blocked_id) VALUES (?, ?)`, blockerID, blockedID)
	return err
}

func (r *UserRepository) UnblockUser(ctx context.Context, blockerID, blockedID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM blocked_users WHERE blocker_id = ? AND blocked_id = ?`, blockerID, blockedID)
	return err
}

func (r *UserRepository) ListBlockedUsers(ctx context.Context, blockerID int64) ([]models.BlockedUser, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT b.id, b.blocker_id, b.blocked_id, u.name, b.created_at
         FROM blocked_users b JOIN users u ON u.id = b.blocked_id
         WHERE b.blocker_id = ? ORDER BY b.created_at DESC`,
		blockerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BlockedUser
	for rows.Next() {
		var b models.BlockedUser
		if err := rows.Scan(&b.ID, &b.BlockerID, &b.BlockedID, &b.BlockedName, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *UserRepository) AreBlocked(ctx context.Context, userA, userB int64) (bool, error) {
	var exists int
	err := r.DB.QueryRowContext(
		ctx,
		`SELECT 1 FROM blocked_users WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?) LIMIT 1`,
		userA, userB, userB, userA,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *UserRepository) CreateNotification(ctx context.Context, userID int64, itemID *int64, title, body string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, ?, ?)`, userID, itemID, title, body)
	return err
}

func (r *UserRepository) ListNotifications(ctx context.Context, userID int64) ([]models.Notification, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT id, user_id, item_id, title, body, read_at, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT 100`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Notification
	for rows.Next() {
		var n models.Notification
		var itemID sql.NullInt64
		var readAt sql.NullTime
		if err := rows.Scan(&n.ID, &n.UserID, &itemID, &n.Title, &n.Body, &readAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		if itemID.Valid {
			v := itemID.Int64
			n.ItemID = &v
		}
		if readAt.Valid {
			v := readAt.Time
			n.ReadAt = &v
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *UserRepository) SaveSearch(ctx context.Context, userID int64, req models.SaveSearchRequest) (models.SavedSearch, error) {
	if req.Name == "" || req.QueryJSON == "" {
		return models.SavedSearch{}, fmt.Errorf("検索条件名と検索条件が必要です")
	}
	result, err := r.DB.ExecContext(ctx, `INSERT INTO saved_searches (user_id, name, query_json) VALUES (?, ?, ?)`, userID, req.Name, req.QueryJSON)
	if err != nil {
		return models.SavedSearch{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.SavedSearch{}, err
	}
	return r.FindSavedSearch(ctx, userID, id)
}

func (r *UserRepository) FindSavedSearch(ctx context.Context, userID, id int64) (models.SavedSearch, error) {
	var s models.SavedSearch
	err := r.DB.QueryRowContext(ctx, `SELECT id, user_id, name, query_json, created_at FROM saved_searches WHERE id = ? AND user_id = ?`, id, userID).Scan(&s.ID, &s.UserID, &s.Name, &s.QueryJSON, &s.CreatedAt)
	return s, err
}

func (r *UserRepository) MarkNotificationRead(ctx context.Context, userID, notificationID int64) (models.Notification, error) {
	_, err := r.DB.ExecContext(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP) WHERE id = ? AND user_id = ?`, notificationID, userID)
	if err != nil {
		return models.Notification{}, err
	}
	var n models.Notification
	var itemID sql.NullInt64
	var readAt sql.NullTime
	err = r.DB.QueryRowContext(ctx, `SELECT id, user_id, item_id, title, body, read_at, created_at FROM notifications WHERE id = ? AND user_id = ?`, notificationID, userID).Scan(&n.ID, &n.UserID, &itemID, &n.Title, &n.Body, &readAt, &n.CreatedAt)
	if err != nil {
		return models.Notification{}, err
	}
	if itemID.Valid {
		v := itemID.Int64
		n.ItemID = &v
	}
	if readAt.Valid {
		v := readAt.Time
		n.ReadAt = &v
	}
	return n, nil
}

func (r *UserRepository) ListSavedSearches(ctx context.Context, userID int64) ([]models.SavedSearch, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, user_id, name, query_json, created_at FROM saved_searches WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.QueryJSON, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *UserRepository) DeleteSavedSearch(ctx context.Context, userID, id int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM saved_searches WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (r *UserRepository) SendSupportMessage(ctx context.Context, userID int64, subject, body string) (models.SupportMessage, error) {
	if subject == "" {
		subject = "一般相談"
	}
	result, err := r.DB.ExecContext(ctx, `INSERT INTO support_messages (user_id, subject, body) VALUES (?, ?, ?)`, userID, subject, body)
	if err != nil {
		return models.SupportMessage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.SupportMessage{}, err
	}
	return r.FindSupportMessage(ctx, userID, id)
}

func (r *UserRepository) FindSupportMessage(ctx context.Context, userID, id int64) (models.SupportMessage, error) {
	var msg models.SupportMessage
	err := r.DB.QueryRowContext(ctx, `SELECT s.id, s.user_id, u.name, COALESCE(s.subject, '一般相談'), s.body, s.created_at FROM support_messages s JOIN users u ON u.id = s.user_id WHERE s.id = ? AND s.user_id = ?`, id, userID).Scan(&msg.ID, &msg.UserID, &msg.UserName, &msg.Subject, &msg.Body, &msg.CreatedAt)
	return msg, err
}

func (r *UserRepository) ListSupportMessages(ctx context.Context, userID int64) ([]models.SupportMessage, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT s.id, s.user_id, u.name, COALESCE(s.subject, '一般相談'), s.body, s.created_at FROM support_messages s JOIN users u ON u.id = s.user_id WHERE s.user_id = ? ORDER BY s.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SupportMessage
	for rows.Next() {
		var msg models.SupportMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.UserName, &msg.Subject, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}
