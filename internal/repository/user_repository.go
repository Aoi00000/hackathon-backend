// ============================================================
// ファイル概要: hackathon-backend/internal/repository/user_repository.go
// 役割: ユーザー登録、認証、プロフィール、残高、通知、支払い方法、保存検索を担当します。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
// Package repository の user_repository は、ユーザー、通知、保存検索条件、ブロック、運営DMを扱います。
//
// マイページに表示する月次/累計の売上・利用額も、この層でDBから集計し、
// フロントエンドが複雑な計算を持たなくてよいようにしています。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hackathon-backend/internal/models"
)

// UserRepository は users とユーザー周辺テーブルへのDB操作を担当します。
// 【詳細コメント】UserRepository は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type UserRepository struct {
	DB *sql.DB
}

// formatJPY は通知本文などのユーザー向け金額を "¥1,200" 形式に整える小さな補助関数です。
// DB/API の内部名には coins が残っていますが、画面・通知では日本円風の表記に統一します。
// 【詳細コメント】formatJPY は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func formatJPY(amount int) string {
	text := strconv.Itoa(amount)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}

// 【詳細コメント】Create は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】scanUser は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func scanUser(scanner interface{ Scan(dest ...any) error }) (models.User, error) {
	// 【詳細コメント】user は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var user models.User
	// 【詳細コメント】ratingAverage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var ratingAverage sql.NullFloat64
	// 【詳細コメント】shippingRegion は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var shippingRegion sql.NullString
	// 【詳細コメント】shippingAddress は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
		&user.MonthlySpendCoins,
		&user.TotalSpendCoins,
		&user.MonthlySalesCoins,
		&user.TotalSalesCoins,
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

// 【詳細コメント】userSelect は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func userSelect(where string) string {
	// 購入・売上の集計は purchases テーブルから毎回算出します。
	// 利用額は「購入手続き完了時点」で購入者の残高から差し引かれるため、canceled以外を数えます。
	// 売上額は「受け取り評価完了時点」で出品者の残高へ反映されるため、completedだけを数えます。
	base := `SELECT u.id, u.name, u.email, u.password_hash, u.balance_coins, u.sales_coins,
                CASE WHEN u.rating_count = 0 THEN 0 ELSE u.rating_sum / u.rating_count END AS rating_average,
                u.rating_count, u.transaction_count, u.shipping_region, u.shipping_address,
                COALESCE((SELECT SUM(p.price_yen) FROM purchases p WHERE p.buyer_id = u.id AND p.status <> 'canceled' AND p.created_at >= DATE_FORMAT(CURRENT_DATE(), '%Y-%m-01')), 0) AS monthly_spend_coins,
                COALESCE((SELECT SUM(p.price_yen) FROM purchases p WHERE p.buyer_id = u.id AND p.status <> 'canceled'), 0) AS total_spend_coins,
                COALESCE((SELECT SUM(p.price_yen) FROM purchases p WHERE p.seller_id = u.id AND p.status = 'completed' AND p.completed_at >= DATE_FORMAT(CURRENT_DATE(), '%Y-%m-01')), 0) AS monthly_sales_coins,
                COALESCE((SELECT SUM(p.price_yen) FROM purchases p WHERE p.seller_id = u.id AND p.status = 'completed'), 0) AS total_sales_coins,
                u.created_at
         FROM users u `
	return base + where
}

// 【詳細コメント】FindByEmail は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	user, err := scanUser(r.DB.QueryRowContext(
		ctx,
		userSelect(`WHERE u.email = ?`),
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

// 【詳細コメント】FindByID は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) FindByID(ctx context.Context, id int64) (models.User, error) {
	user, err := scanUser(r.DB.QueryRowContext(
		ctx,
		userSelect(`WHERE u.id = ?`),
		id,
	))
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

// Charge はアプリ内仮想通貨をチャージします。
// 【詳細コメント】Charge は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) Charge(ctx context.Context, userID int64, amount int) (models.User, error) {
	if amount <= 0 {
		return models.User{}, fmt.Errorf("チャージ金額は1以上にしてください")
	}
	if amount > 1000000 {
		return models.User{}, fmt.Errorf("一度にチャージできる上限を超えています")
	}
	hasDefault, err := r.HasDefaultPaymentMethod(ctx, userID)
	if err != nil {
		return models.User{}, err
	}
	if !hasDefault {
		return models.User{}, fmt.Errorf("残高チャージには、使用する支払い方法を1つ以上登録し、既定に設定してください")
	}
	if _, err := r.DB.ExecContext(ctx, `UPDATE users SET balance_coins = balance_coins + ? WHERE id = ?`, amount, userID); err != nil {
		return models.User{}, err
	}
	// チャージはユーザーにとって重要な残高変動なので、通知一覧にも記録します。
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, 'チャージ完了', ?)`, userID, fmt.Sprintf("¥%sをチャージしました", formatJPY(amount)))
	return r.FindByID(ctx, userID)
}

// 【詳細コメント】UpdateProfile は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】BlockUser は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) BlockUser(ctx context.Context, blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return fmt.Errorf("自分自身はブロックできません")
	}
	_, err := r.DB.ExecContext(ctx, `INSERT IGNORE INTO blocked_users (blocker_id, blocked_id) VALUES (?, ?)`, blockerID, blockedID)
	return err
}

// 【詳細コメント】UnblockUser は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) UnblockUser(ctx context.Context, blockerID, blockedID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM blocked_users WHERE blocker_id = ? AND blocked_id = ?`, blockerID, blockedID)
	return err
}

// 【詳細コメント】ListBlockedUsers は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
	// 【詳細コメント】out は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var out []models.BlockedUser
	for rows.Next() {
		// 【詳細コメント】b は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var b models.BlockedUser
		if err := rows.Scan(&b.ID, &b.BlockerID, &b.BlockedID, &b.BlockedName, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// 【詳細コメント】AreBlocked は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) AreBlocked(ctx context.Context, userA, userB int64) (bool, error) {
	// 【詳細コメント】exists は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】CreateNotification は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) CreateNotification(ctx context.Context, userID int64, itemID *int64, title, body string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, ?, ?)`, userID, itemID, title, body)
	return err
}

// 【詳細コメント】ListNotifications は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
	// 【詳細コメント】out は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var out []models.Notification
	for rows.Next() {
		// 【詳細コメント】n は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var n models.Notification
		// 【詳細コメント】itemID は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var itemID sql.NullInt64
		// 【詳細コメント】readAt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】SaveSearch は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】FindSavedSearch は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) FindSavedSearch(ctx context.Context, userID, id int64) (models.SavedSearch, error) {
	// 【詳細コメント】s は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var s models.SavedSearch
	err := r.DB.QueryRowContext(ctx, `SELECT id, user_id, name, query_json, created_at FROM saved_searches WHERE id = ? AND user_id = ?`, id, userID).Scan(&s.ID, &s.UserID, &s.Name, &s.QueryJSON, &s.CreatedAt)
	return s, err
}

// 【詳細コメント】MarkNotificationRead は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) MarkNotificationRead(ctx context.Context, userID, notificationID int64) (models.Notification, error) {
	_, err := r.DB.ExecContext(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP) WHERE id = ? AND user_id = ?`, notificationID, userID)
	if err != nil {
		return models.Notification{}, err
	}
	// 【詳細コメント】n は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var n models.Notification
	// 【詳細コメント】itemID は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var itemID sql.NullInt64
	// 【詳細コメント】readAt は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】ListSavedSearches は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) ListSavedSearches(ctx context.Context, userID int64) ([]models.SavedSearch, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, user_id, name, query_json, created_at FROM saved_searches WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 【詳細コメント】out は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var out []models.SavedSearch
	for rows.Next() {
		// 【詳細コメント】s は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var s models.SavedSearch
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.QueryJSON, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// 【詳細コメント】DeleteSavedSearch は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) DeleteSavedSearch(ctx context.Context, userID, id int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM saved_searches WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// 【詳細コメント】SendSupportMessage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】FindSupportMessage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) FindSupportMessage(ctx context.Context, userID, id int64) (models.SupportMessage, error) {
	// 【詳細コメント】msg は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var msg models.SupportMessage
	err := r.DB.QueryRowContext(ctx, `SELECT s.id, s.user_id, u.name, COALESCE(s.subject, '一般相談'), s.body, s.created_at FROM support_messages s JOIN users u ON u.id = s.user_id WHERE s.id = ? AND s.user_id = ?`, id, userID).Scan(&msg.ID, &msg.UserID, &msg.UserName, &msg.Subject, &msg.Body, &msg.CreatedAt)
	return msg, err
}

// 【詳細コメント】ListSupportMessages は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) ListSupportMessages(ctx context.Context, userID int64) ([]models.SupportMessage, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT s.id, s.user_id, u.name, COALESCE(s.subject, '一般相談'), s.body, s.created_at FROM support_messages s JOIN users u ON u.id = s.user_id WHERE s.user_id = ? ORDER BY s.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 【詳細コメント】out は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var out []models.SupportMessage
	for rows.Next() {
		// 【詳細コメント】msg は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var msg models.SupportMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.UserName, &msg.Subject, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

// 【詳細コメント】maskCardNumber は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func maskCardNumber(cardNumber string) string {
	cleaned := strings.NewReplacer(" ", "", "-", "").Replace(cardNumber)
	if len(cleaned) < 4 {
		return ""
	}
	return cleaned[len(cleaned)-4:]
}

// 【詳細コメント】ListMonthlyMoneySummary は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) ListMonthlyMoneySummary(ctx context.Context, userID int64, months int) ([]models.MonthlyMoneySummary, error) {
	if months <= 0 || months > 24 {
		months = 6
	}
	now := time.Now().UTC()
	firstMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(months - 1), 0)

	// まず直近monthsか月の箱を0円で作っておきます。
	// 取引がない月もグラフに出すことで、デモ時に「空白の月」が分かりやすくなります。
	indexByMonth := map[string]int{}
	out := make([]models.MonthlyMoneySummary, months)
	for i := 0; i < months; i++ {
		m := firstMonth.AddDate(0, i, 0).Format("2006-01")
		out[i] = models.MonthlyMoneySummary{Month: m}
		indexByMonth[m] = i
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT ym,
		       SUM(sales_yen) AS sales_yen,
		       SUM(spend_yen) AS spend_yen
		FROM (
			SELECT DATE_FORMAT(completed_at, '%Y-%m') AS ym,
			       price_yen AS sales_yen,
			       0 AS spend_yen
			FROM purchases
			WHERE seller_id = ?
			  AND status = 'completed'
			  AND completed_at >= ?
			UNION ALL
			SELECT DATE_FORMAT(created_at, '%Y-%m') AS ym,
			       0 AS sales_yen,
			       price_yen AS spend_yen
			FROM purchases
			WHERE buyer_id = ?
			  AND status <> 'canceled'
			  AND created_at >= ?
		) x
		GROUP BY ym
		ORDER BY ym ASC`, userID, firstMonth, userID, firstMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		// 【詳細コメント】month は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var month string
		// 【詳細コメント】sales は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var sales, spend sql.NullInt64
		if err := rows.Scan(&month, &sales, &spend); err != nil {
			return nil, err
		}
		if idx, ok := indexByMonth[month]; ok {
			if sales.Valid {
				out[idx].SalesYen = int(sales.Int64)
			}
			if spend.Valid {
				out[idx].SpendYen = int(spend.Int64)
			}
		}
	}
	return out, rows.Err()
}

// 【詳細コメント】ListPaymentMethods は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) ListPaymentMethods(ctx context.Context, userID int64) ([]models.PaymentMethod, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, user_id, label, card_last4, holder_name, expiry_month, expiry_year, is_default, created_at FROM payment_methods WHERE user_id = ? ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.PaymentMethod{}
	for rows.Next() {
		// 【詳細コメント】m は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var m models.PaymentMethod
		// 【詳細コメント】isDefault は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var isDefault int
		if err := rows.Scan(&m.ID, &m.UserID, &m.Label, &m.CardLast4, &m.HolderName, &m.ExpiryMonth, &m.ExpiryYear, &isDefault, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.IsDefault = isDefault == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

// 【詳細コメント】CreatePaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) CreatePaymentMethod(ctx context.Context, userID int64, req models.CreatePaymentMethodRequest) (models.PaymentMethod, error) {
	req.Label = strings.TrimSpace(req.Label)
	req.HolderName = strings.TrimSpace(req.HolderName)
	last4 := maskCardNumber(req.CardNumber)
	securityCode := strings.TrimSpace(req.SecurityCode)
	if req.Label == "" || req.HolderName == "" || last4 == "" || req.ExpiryMonth < 1 || req.ExpiryMonth > 12 || req.ExpiryYear < time.Now().Year()%100 || len(securityCode) < 3 {
		return models.PaymentMethod{}, fmt.Errorf("登録名、カード番号、名義、有効期限、セキュリティコードを正しく入力してください")
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.PaymentMethod{}, err
	}
	defer tx.Rollback()

	// 【詳細コメント】existing は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_methods WHERE user_id = ?`, userID).Scan(&existing); err != nil {
		return models.PaymentMethod{}, err
	}
	isDefault := req.IsDefault || existing == 0
	if isDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE payment_methods SET is_default = 0 WHERE user_id = ?`, userID); err != nil {
			return models.PaymentMethod{}, err
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO payment_methods (user_id, label, card_last4, holder_name, expiry_month, expiry_year, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, req.Label, last4, req.HolderName, req.ExpiryMonth, req.ExpiryYear, isDefault)
	if err != nil {
		return models.PaymentMethod{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.PaymentMethod{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, '支払い方法登録完了', ?)`, userID, fmt.Sprintf("%s（下4桁 %s）を支払い方法として登録しました", req.Label, last4)); err != nil {
		return models.PaymentMethod{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.PaymentMethod{}, err
	}
	return r.FindPaymentMethod(ctx, userID, id)
}

// 【詳細コメント】FindPaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) FindPaymentMethod(ctx context.Context, userID, id int64) (models.PaymentMethod, error) {
	// 【詳細コメント】m は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var m models.PaymentMethod
	// 【詳細コメント】isDefault は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var isDefault int
	err := r.DB.QueryRowContext(ctx, `SELECT id, user_id, label, card_last4, holder_name, expiry_month, expiry_year, is_default, created_at FROM payment_methods WHERE id = ? AND user_id = ?`, id, userID).Scan(&m.ID, &m.UserID, &m.Label, &m.CardLast4, &m.HolderName, &m.ExpiryMonth, &m.ExpiryYear, &isDefault, &m.CreatedAt)
	m.IsDefault = isDefault == 1
	return m, err
}

// 【詳細コメント】SetDefaultPaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) SetDefaultPaymentMethod(ctx context.Context, userID, id int64) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 【詳細コメント】exists は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_methods WHERE id = ? AND user_id = ?`, id, userID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("支払い方法が見つかりません")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payment_methods SET is_default = 0 WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payment_methods SET is_default = 1 WHERE id = ? AND user_id = ?`, id, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, '支払い方法変更完了', '残高チャージに使用する既定の支払い方法を変更しました')`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// 【詳細コメント】DeletePaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) DeletePaymentMethod(ctx context.Context, userID, id int64) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 【詳細コメント】wasDefault は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var wasDefault int
	if err := tx.QueryRowContext(ctx, `SELECT is_default FROM payment_methods WHERE id = ? AND user_id = ?`, id, userID).Scan(&wasDefault); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM payment_methods WHERE id = ? AND user_id = ?`, id, userID); err != nil {
		return err
	}
	if wasDefault == 1 {
		// 【詳細コメント】nextID は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
		var nextID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM payment_methods WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`, userID).Scan(&nextID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			if _, err := tx.ExecContext(ctx, `UPDATE payment_methods SET is_default = 1 WHERE id = ? AND user_id = ?`, nextID, userID); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, NULL, '支払い方法削除完了', '登録済みの支払い方法を削除しました')`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// 【詳細コメント】HasDefaultPaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func (r *UserRepository) HasDefaultPaymentMethod(ctx context.Context, userID int64) (bool, error) {
	// 【詳細コメント】exists は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
	var exists int
	err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM payment_methods WHERE user_id = ? AND is_default = 1 LIMIT 1`, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
