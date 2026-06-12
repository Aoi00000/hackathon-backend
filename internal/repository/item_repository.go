package repository

import (
	"context"
	"database/sql"
	"fmt"

	"hackathon-backend/internal/models"
)

// ItemRepository は items テーブルへのDB操作を担当します。
type ItemRepository struct {
	DB *sql.DB
}

// List は商品一覧を取得します。
// qが空でなければタイトル・説明・カテゴリに対してLIKE検索を行います。
func (r *ItemRepository) List(ctx context.Context, q string) ([]models.Item, error) {
	base := `SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                   i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                   i.created_at, i.updated_at
            FROM items i
            JOIN users u ON u.id = i.seller_id`
	args := []any{}

	if q != "" {
		base += ` WHERE i.title LIKE ? OR i.description LIKE ? OR i.category LIKE ?`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}

	base += ` ORDER BY i.created_at DESC LIMIT 100`

	rows, err := r.DB.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(
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
		); err != nil {
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

// FindByID は商品詳細を取得します。
func (r *ItemRepository) FindByID(ctx context.Context, id int64) (models.Item, error) {
	var item models.Item
	err := r.DB.QueryRowContext(
		ctx,
		`SELECT i.id, i.seller_id, u.name, i.title, i.description, i.category,
                i.condition_text, i.price_yen, COALESCE(i.image_url, ''), i.status,
                i.created_at, i.updated_at
         FROM items i
         JOIN users u ON u.id = i.seller_id
         WHERE i.id = ?`,
		id,
	).Scan(
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

	if err != nil {
		return models.Item{}, err
	}
	return item, nil
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

	if _, err := tx.ExecContext(ctx, `UPDATE items SET status = 'sold' WHERE id = ?`, itemID); err != nil {
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

	return models.Purchase{
		ID:       purchaseID,
		ItemID:   itemID,
		BuyerID:  buyerID,
		SellerID: sellerID,
		PriceYen: priceYen,
	}, nil
}
