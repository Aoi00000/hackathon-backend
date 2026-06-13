package repository

import (
	"context"
	"database/sql"
	"fmt"

	"hackathon-backend/internal/models"
)

// ItemRepository は items / purchases / checklist テーブルへのDB操作を担当します。
// MVPではリポジトリを細かく分けすぎず、商品に強く関係する処理をここに集約しています。
type ItemRepository struct {
	DB *sql.DB
}

// scanItem は商品取得SQLで共通利用するScan処理です。
// SELECT句の順序を揃えることで、List / FindByID / ListBySeller で同じ読み取り処理を使えます。
func scanItem(scanner interface{ Scan(dest ...any) error }) (models.Item, error) {
	var item models.Item
	err := scanner.Scan(
		&item.ID,
		&item.SellerID,
		&item.SellerName,
		&item.Title,
		&item.Description,
		&item.Category,
		&item.ConditionText,
		&item.PriceYen,
		&item.ImageURL,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

// List は商品一覧を取得します。
// キャンセル済みの商品は一般の商品一覧には出さないことで、購入者に不要な商品を見せないようにします。
func (r *ItemRepository) List(ctx context.Context, q string) ([]models.Item, error) {
	base := `SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                   i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                   i.created_at, i.updated_at
            FROM items i
            JOIN users u ON u.id = i.seller_id
            WHERE i.status <> 'canceled'`
	args := []any{}

	if q != "" {
		base += ` AND (i.title LIKE ? OR i.description LIKE ? OR i.category LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}

	base += ` ORDER BY i.updated_at DESC LIMIT 100`

	rows, err := r.DB.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// ListBySeller はログイン中ユーザーの出品履歴を取得します。
// 出品キャンセル済みの商品も履歴として見たいので、ここではcanceledも含めます。
func (r *ItemRepository) ListBySeller(ctx context.Context, sellerID int64) ([]models.Item, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                i.created_at, i.updated_at
         FROM items i
         JOIN users u ON u.id = i.seller_id
         WHERE i.seller_id = ?
         ORDER BY i.updated_at DESC`,
		sellerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Create は商品を出品します。
func (r *ItemRepository) Create(ctx context.Context, sellerID int64, req models.CreateItemRequest) (models.Item, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO items (seller_id, title, description, category, condition_text, price_yen, image_url)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sellerID,
		req.Title,
		req.Description,
		req.Category,
		req.ConditionText,
		req.PriceYen,
		req.ImageURL,
	)
	if err != nil {
		return models.Item{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Item{}, err
	}

	return r.FindByID(ctx, id)
}

// Update は出品者本人が商品情報を編集します。
// sold / canceled の商品は履歴として残すことを優先し、編集不可にしています。
func (r *ItemRepository) Update(ctx context.Context, itemID, sellerID int64, req models.UpdateItemRequest) (models.Item, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`UPDATE items
         SET title = ?, description = ?, category = ?, condition_text = ?, price_yen = ?, image_url = ?, updated_at = CURRENT_TIMESTAMP
         WHERE id = ? AND seller_id = ? AND status = 'available'`,
		req.Title,
		req.Description,
		req.Category,
		req.ConditionText,
		req.PriceYen,
		req.ImageURL,
		itemID,
		sellerID,
	)
	if err != nil {
		return models.Item{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Item{}, err
	}
	if affected == 0 {
		return models.Item{}, fmt.Errorf("商品が見つからないか、編集できない状態です")
	}

	return r.FindByID(ctx, itemID)
}

// Cancel は出品者本人が出品をキャンセルします。
// 購入済みの商品はキャンセルできないよう、availableのときだけcanceledへ変更します。
func (r *ItemRepository) Cancel(ctx context.Context, itemID, sellerID int64) (models.Item, error) {
	result, err := r.DB.ExecContext(
		ctx,
		`UPDATE items
         SET status = 'canceled', updated_at = CURRENT_TIMESTAMP
         WHERE id = ? AND seller_id = ? AND status = 'available'`,
		itemID,
		sellerID,
	)
	if err != nil {
		return models.Item{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Item{}, err
	}
	if affected == 0 {
		return models.Item{}, fmt.Errorf("商品が見つからないか、キャンセルできない状態です")
	}

	return r.FindByID(ctx, itemID)
}

// FindByID は商品詳細を取得します。
func (r *ItemRepository) FindByID(ctx context.Context, id int64) (models.Item, error) {
	return scanItem(r.DB.QueryRowContext(
		ctx,
		`SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                i.created_at, i.updated_at
         FROM items i
         JOIN users u ON u.id = i.seller_id
         WHERE i.id = ?`,
		id,
	))
}

// Purchase は商品購入をトランザクションで行います。
// 商品の状態確認、soldへの更新、purchasesへの記録を一つの単位として扱うことで、二重購入を防ぎます。
func (r *ItemRepository) Purchase(ctx context.Context, itemID, buyerID int64) (models.Purchase, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Purchase{}, err
	}
	defer tx.Rollback()

	var sellerID int64
	var priceYen int
	var status string

	// FOR UPDATE で対象商品行をロックし、同時購入の競合を防ぎます。
	err = tx.QueryRowContext(
		ctx,
		`SELECT seller_id, price_yen, status FROM items WHERE id = ? FOR UPDATE`,
		itemID,
	).Scan(&sellerID, &priceYen, &status)
	if err != nil {
		return models.Purchase{}, err
	}

	if sellerID == buyerID {
		return models.Purchase{}, fmt.Errorf("自分の商品は購入できません")
	}
	if status != "available" {
		return models.Purchase{}, fmt.Errorf("この商品は購入できません")
	}

	if _, err := tx.ExecContext(ctx, `UPDATE items SET status = 'sold', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, itemID); err != nil {
		return models.Purchase{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO purchases (item_id, buyer_id, seller_id, price_yen) VALUES (?, ?, ?, ?)`,
		itemID,
		buyerID,
		sellerID,
		priceYen,
	)
	if err != nil {
		return models.Purchase{}, err
	}

	purchaseID, err := result.LastInsertId()
	if err != nil {
		return models.Purchase{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Purchase{}, err
	}

	return models.Purchase{ID: purchaseID, ItemID: itemID, BuyerID: buyerID, SellerID: sellerID, PriceYen: priceYen}, nil
}

// ListPurchasesByBuyer は購入履歴を購入日時の新しい順に取得します。
func (r *ItemRepository) ListPurchasesByBuyer(ctx context.Context, buyerID int64) ([]models.PurchaseHistory, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT p.id, i.id, i.seller_id, u.name, i.title, i.description, i.category, i.condition_text,
                p.price_yen, COALESCE(i.image_url, ''), i.status, p.created_at, i.updated_at
         FROM purchases p
         JOIN items i ON i.id = p.item_id
         JOIN users u ON u.id = i.seller_id
         WHERE p.buyer_id = ?
         ORDER BY p.created_at DESC`,
		buyerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []models.PurchaseHistory{}
	for rows.Next() {
		var row models.PurchaseHistory
		if err := rows.Scan(
			&row.PurchaseID,
			&row.ItemID,
			&row.SellerID,
			&row.SellerName,
			&row.Title,
			&row.Description,
			&row.Category,
			&row.ConditionText,
			&row.PriceYen,
			&row.ImageURL,
			&row.Status,
			&row.PurchasedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, row)
	}
	return history, rows.Err()
}

// ListChecklist はログイン中ユーザーのチェックリスト商品を取得します。
func (r *ItemRepository) ListChecklist(ctx context.Context, userID int64) ([]models.Item, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                i.created_at, i.updated_at
         FROM checklist c
         JOIN items i ON i.id = c.item_id
         JOIN users u ON u.id = i.seller_id
         WHERE c.user_id = ? AND i.status <> 'canceled'
         ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// IsInChecklist は指定商品がユーザーのチェックリストに入っているかを返します。
func (r *ItemRepository) IsInChecklist(ctx context.Context, userID, itemID int64) (bool, error) {
	var exists int
	err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM checklist WHERE user_id = ? AND item_id = ?`, userID, itemID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// AddChecklist は商品をチェックリストに追加します。
func (r *ItemRepository) AddChecklist(ctx context.Context, userID, itemID int64) error {
	_, err := r.DB.ExecContext(
		ctx,
		`INSERT IGNORE INTO checklist (user_id, item_id) VALUES (?, ?)`,
		userID,
		itemID,
	)
	return err
}

// RemoveChecklist は商品をチェックリストから外します。
func (r *ItemRepository) RemoveChecklist(ctx context.Context, userID, itemID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM checklist WHERE user_id = ? AND item_id = ?`, userID, itemID)
	return err
}
