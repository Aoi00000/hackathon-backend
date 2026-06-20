// ============================================================
// ファイル概要: hackathon-backend/internal/repository/item_repository.go
// 役割: 商品、購入、チェックリスト、月別集計、AI販売改善通知など商品中心のDB処理を担当します。
//
// ============================================================
// 実装詳細メモ:
// 商品検索、出品、購入、発送、受取完了、チェックリスト、推薦、売れ残り通知を扱う中心的なRepositoryです。
// 在庫状態と購入状態を同じトランザクションで更新し、二重購入や残高不整合を防ぎます。
// Package repository の item_repository は、商品・購入・チェックリスト・推薦に関するDB操作を担当します。
//
// 購入や出品キャンセルなど、複数テーブルを同時に更新する処理はトランザクションで扱います。
// これにより、残高だけ減った、通知だけ作られた、といった中途半端な状態を避けます。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"hackathon-backend/internal/models"
)

// ItemRepository は、商品・購入・チェックリストなど「商品を中心に動く機能」のDB窓口です。
// Handler層はHTTPリクエストを読む役割に集中し、SQLやトランザクションの詳細はこのRepositoryに閉じ込めます。
type ItemRepository struct{ DB *sql.DB }

// ItemFilter は、商品一覧画面の検索フォームや自然言語検索結果をSQL条件へ渡すための入れ物です。
// 文字列の項目は未指定なら空文字、価格は未指定なら0として扱い、Repository側で必要な条件だけを組み立てます。
type ItemFilter struct {
	Query          string
	Category       string
	Size           string
	Color          string
	ConditionText  string
	Status         string
	MinPrice       int
	MaxPrice       int
	Tag            string
	Sort           string
	DeliveryWithin string
}

// scanItem は、itemSelect が返すSELECT列を models.Item へ詰め替える共通関数です。
// SQLのLEFT JOINでは購入者や購入情報が存在しない商品も返るため、NullString/NullInt64/NullTimeを使って
// 「DB上はNULLだが、APIレスポンスでは空文字やnilポインタにしたい値」を丁寧に変換します。
func scanItem(scanner interface{ Scan(dest ...any) error }) (models.Item, error) {
	var item models.Item
	var productCode, imageURL, deliveryMethod, shipFromRegion, size, color, tags, buyerName, buyerAddress, purchaseStatus sql.NullString
	var buyerID, purchaseID sql.NullInt64
	var sellerRating sql.NullFloat64
	var purchaseCreatedAt, shippingDeadline, shippedAt, completedAt sql.NullTime
	err := scanner.Scan(
		&item.ID, &productCode, &item.SellerID, &item.SellerName, &sellerRating, &item.SellerRatingCount, &item.SellerTransactionCount,
		&item.Title, &item.Description, &item.Category, &item.ConditionText, &item.PriceYen, &imageURL, &item.Status,
		&deliveryMethod, &item.ShippingDays, &shipFromRegion, &size, &color, &tags, &item.ChecklistCount,
		&buyerID, &buyerName, &buyerAddress, &purchaseID, &purchaseStatus, &purchaseCreatedAt, &shippingDeadline, &shippedAt, &completedAt,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if productCode.Valid {
		item.ProductCode = productCode.String
	}
	if imageURL.Valid {
		item.ImageURL = imageURL.String
	}
	if deliveryMethod.Valid {
		item.DeliveryMethod = deliveryMethod.String
	}
	if shipFromRegion.Valid {
		item.ShipFromRegion = shipFromRegion.String
	}
	if size.Valid {
		item.Size = size.String
	}
	if color.Valid {
		item.Color = color.String
	}
	if tags.Valid {
		item.Tags = tags.String
	}
	if sellerRating.Valid {
		item.SellerRatingAverage = sellerRating.Float64
	}
	if buyerID.Valid {
		v := buyerID.Int64
		item.BuyerID = &v
	}
	if buyerName.Valid {
		item.BuyerName = buyerName.String
	}
	if buyerAddress.Valid {
		item.BuyerShippingAddress = buyerAddress.String
	}
	if purchaseID.Valid {
		v := purchaseID.Int64
		item.PurchaseID = &v
	}
	if purchaseStatus.Valid {
		item.PurchaseStatus = purchaseStatus.String
	}
	if purchaseCreatedAt.Valid {
		v := purchaseCreatedAt.Time
		item.PurchaseCreatedAt = &v
	}
	if shippingDeadline.Valid {
		v := shippingDeadline.Time
		item.ShippingDeadline = &v
	}
	if shippedAt.Valid {
		v := shippedAt.Time
		item.ShippedAt = &v
	}
	if completedAt.Valid {
		v := completedAt.Time
		item.CompletedAt = &v
	}
	return item, err
}

// itemSelect は、商品一覧・詳細・マイページ出品一覧などで共通利用するSELECT句です。
// ここで列の並びを一箇所に固定し、scanItemのScan順と対応させることで、同じ商品情報をどの画面でも同じ形で取得します。
func itemSelect() string {
	return `SELECT i.id, i.product_code, i.seller_id, u.name,
                CASE WHEN u.rating_count = 0 THEN 0 ELSE u.rating_sum / u.rating_count END AS seller_rating_average,
                u.rating_count, u.transaction_count,
                i.title, i.description, i.category, i.condition_text, i.price_yen, i.image_url, i.status,
                i.delivery_method, i.shipping_days, i.ship_from_region, i.size, i.color, i.tags,
                (SELECT COUNT(*) FROM checklist c WHERE c.item_id = i.id) AS checklist_count,
                p.buyer_id, buyer.name, p.delivery_address, p.id, p.status, p.created_at, p.shipping_deadline, p.shipped_at, p.completed_at,
                i.created_at, i.updated_at
         FROM items i
         JOIN users u ON u.id = i.seller_id
         LEFT JOIN purchases p ON p.item_id = i.id
         LEFT JOIN users buyer ON buyer.id = p.buyer_id`
}

// splitFilterValues は、URLクエリのカンマ区切り複数選択をSQL用の値配列へ変換します。
// 例として category=本,家電 のような入力を ["本", "家電"] にし、空白や空要素は検索条件から外します。
func splitFilterValues(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// addInFilter は、カテゴリ・サイズ・色のような複数選択フィルタを安全にSQLへ追加します。
// 値を文字列連結で直接埋め込まず、? プレースホルダと args を使うことでSQLインジェクションを避けます。
func addInFilter(query *string, args *[]any, column string, raw string) {
	values := splitFilterValues(raw)
	if len(values) == 0 {
		return
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	*query += fmt.Sprintf(" AND %s IN (%s)", column, placeholders)
	for _, v := range values {
		*args = append(*args, v)
	}
}

// normalizeKanaJP は、日本語検索でよく起きる「カタカナとひらがなの表記揺れ」を吸収します。
// フリマの商品名では「ギター」「ぎたー」のような入力差があるため、曖昧検索の前処理として使います。
func normalizeKanaJP(value string) string {
	// カタカナをひらがなに寄せます。
	// 例: ギター -> ぎたー、タマネギ -> たまねぎ。
	runes := []rune(value)
	for i, r := range runes {
		if r >= 'ァ' && r <= 'ヶ' {
			runes[i] = r - 0x60
		}
	}
	return string(runes)
}

// normalizeJP は、商品名や検索語を比較しやすい標準形に寄せるための関数です。
// SQLのLIKEだけでは「教科書」と「参考書」、「スマートフォン」と「スマホ」のような意味の近さを拾いにくいため、
// アプリ内で想定される代表的な言い換えを辞書化してから空白・記号を取り除きます。
func normalizeJP(value string) string {
	// DB検索でSQLだけに頼ると、漢字/ひらがな/カタカナの表記揺れを拾いにくくなります。
	// 依存を増やさずに動かすため、代表的な語の読みを辞書化し、
	// その後に記号や空白を除去して曖昧検索に使います。
	normalized := normalizeKanaJP(value)
	replacer := strings.NewReplacer(
		" ", "", "　", "", "-", "", "_", "", "/", "", "・", "", ",", "", "，", "", ".", "", "．", "", "、", "", "。", "", "(", "", ")", "", "（", "", "）", "",
		"玉葱", "玉ねぎ", "たまねぎ", "玉ねぎ", "onion", "玉ねぎ",
		"人参", "にんじん", "carrot", "にんじん",
		"馬鈴薯", "じゃがいも", "potato", "じゃがいも",
		"食べ物", "食品", "フード", "食品",
		"教科書", "参考書", "教材", "参考書", "書籍", "本",
		"スマートフォン", "スマホ", "携帯", "スマホ",
		"数学", "すうがく", "算数", "すうがく", "math", "すうがく",
		"ギター", "ぎたー", "guitar", "ぎたー", "エレキギター", "ぎたー", "アコギ", "ぎたー",
		"大学受験", "受験", "入試", "受験",
	)
	return strings.ToLower(replacer.Replace(normalized))
}

// levenshtein は、2つの文字列が何文字分違うかを測る編集距離アルゴリズムです。
// 1文字の入力ミスや表記ぶれを許容した検索に使い、短いクエリでも「完全一致しないから0件」を減らします。
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	dp := make([][]int, len(ra)+1)
	for i := range dp {
		dp[i] = make([]int, len(rb)+1)
		dp[i][0] = i
	}
	for j := 0; j <= len(rb); j++ {
		dp[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[len(ra)][len(rb)]
}

// fuzzyMatchItem は、DBから広めに取得した商品に対してGo側で柔軟なキーワード判定を行います。
// 商品名・説明・カテゴリ・状態・タグ・出品者名をまとめて正規化し、部分一致または編集距離1以内なら一致とみなします。
func fuzzyMatchItem(item models.Item, query string) bool {
	q := normalizeJP(query)
	if q == "" {
		return true
	}
	target := normalizeJP(strings.Join([]string{item.Title, item.Description, item.Category, item.ConditionText, item.Size, item.Color, item.Tags, item.SellerName}, " "))
	if strings.Contains(target, q) {
		return true
	}
	qr := []rune(q)
	tr := []rune(target)
	if len(qr) >= 3 && len(tr) >= len(qr) {
		for i := 0; i+len(qr) <= len(tr); i++ {
			if levenshtein(string(tr[i:i+len(qr)]), q) <= 1 {
				return true
			}
		}
	}
	return false
}

// List は、商品一覧ページと自然言語検索の中心になる検索処理です。
// SQLで確実に絞れる条件をDBに任せ、表記揺れを含むキーワード検索はfuzzyMatchItemで補完します。
func (r *ItemRepository) List(ctx context.Context, f ItemFilter, viewerID *int64) ([]models.Item, error) {
	query := itemSelect() + ` WHERE i.status <> 'canceled'`
	args := []any{}
	if viewerID != nil {
		query += ` AND NOT EXISTS (SELECT 1 FROM blocked_users b WHERE (b.blocker_id = ? AND b.blocked_id = i.seller_id) OR (b.blocker_id = i.seller_id AND b.blocked_id = ?))`
		args = append(args, *viewerID, *viewerID)
	}
	// キーワード検索は、漢字/ひらがな/表記揺れをGo側で柔軟に判定します。
	// そのためSQLでは絞り込みすぎず、カテゴリなど確実な条件だけDBで絞ります。
	addInFilter(&query, &args, "i.category", f.Category)
	addInFilter(&query, &args, "i.size", f.Size)
	addInFilter(&query, &args, "i.color", f.Color)
	addInFilter(&query, &args, "i.condition_text", f.ConditionText)
	addInFilter(&query, &args, "i.status", f.Status)
	if f.MinPrice > 0 {
		query += ` AND i.price_yen >= ?`
		args = append(args, f.MinPrice)
	}
	if f.MaxPrice > 0 {
		query += ` AND i.price_yen <= ?`
		args = append(args, f.MaxPrice)
	}
	if f.Tag != "" {
		query += ` AND i.tags LIKE ?`
		args = append(args, "%"+f.Tag+"%")
	}

	// 発送までの日数は、実際には「発送までの日数」を簡易的に近似して検索します。
	// 画面上では「本日中」「明日中」などユーザーに分かりやすい言葉を使い、
	// DB上では shipping_days の上限/下限に変換します。
	switch f.DeliveryWithin {
	case "today", "tomorrow":
		query += ` AND i.shipping_days <= 1`
	case "3days":
		query += ` AND i.shipping_days <= 3`
	case "week":
		query += ` AND i.shipping_days <= 7`
	case "later":
		query += ` AND i.shipping_days > 7`
	}
	switch f.Sort {
	case "price_asc":
		query += ` ORDER BY i.price_yen ASC, i.updated_at DESC`
	case "price_desc":
		query += ` ORDER BY i.price_yen DESC, i.updated_at DESC`
	case "checklist_desc":
		query += ` ORDER BY checklist_count DESC, i.updated_at DESC`
	case "recommended":
		query += ` ORDER BY (CASE WHEN i.status='available' THEN 0 ELSE 1 END), checklist_count DESC, i.updated_at DESC`
	default:
		query += ` ORDER BY i.updated_at DESC`
	}
	query += ` LIMIT 300`
	rows, err := r.DB.QueryContext(ctx, query, args...)
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
		if fuzzyMatchItem(item, f.Query) {
			items = append(items, item)
		}
	}
	if len(items) > 100 {
		items = items[:100]
	}
	return items, rows.Err()
}

// ListBySeller は、マイページの「自分の出品」一覧を取得します。
// 商品ステータスに関係なく出品者IDで取得するため、販売中・売却済み・キャンセル済みの履歴を同じ画面で確認できます。
func (r *ItemRepository) ListBySeller(ctx context.Context, sellerID int64) ([]models.Item, error) {
	rows, err := r.DB.QueryContext(ctx, itemSelect()+` WHERE i.seller_id = ? ORDER BY i.updated_at DESC`, sellerID)
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

// Create は、新規出品をDBへ保存し、出品完了通知と商品コードを作成します。
// DBの自動採番IDが確定した後で AFM-000001 のような表示用コードを生成するため、INSERT後にUPDATEしています。
func (r *ItemRepository) Create(ctx context.Context, sellerID int64, req models.CreateItemRequest) (models.Item, error) {
	if req.DeliveryMethod == "" {
		req.DeliveryMethod = "対面・配送相談"
	}
	if req.ShippingDays <= 0 {
		req.ShippingDays = 2
	}
	if req.ShipFromRegion == "" {
		req.ShipFromRegion = "未設定"
	}
	result, err := r.DB.ExecContext(ctx,
		`INSERT INTO items (seller_id, title, description, category, condition_text, price_yen, image_url, delivery_method, shipping_days, ship_from_region, size, color, tags)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sellerID, req.Title, req.Description, req.Category, req.ConditionText, req.PriceYen, req.ImageURL, req.DeliveryMethod, req.ShippingDays, req.ShipFromRegion, req.Size, req.Color, req.Tags)
	if err != nil {
		return models.Item{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Item{}, err
	}
	code := fmt.Sprintf("AFM-%06d", id)
	if _, err := r.DB.ExecContext(ctx, `UPDATE items SET product_code = ? WHERE id = ?`, code, id); err != nil {
		return models.Item{}, err
	}
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '出品完了', '商品の出品が完了しました')`, sellerID, id)
	return r.FindByID(ctx, id)
}

// Update は、出品者本人が販売中の商品情報だけを編集する処理です。
// すでに購入された商品を後から変更できると取引条件が変わってしまうため、status='available' の場合に限定します。
func (r *ItemRepository) Update(ctx context.Context, itemID, sellerID int64, req models.UpdateItemRequest) (models.Item, error) {
	if req.DeliveryMethod == "" {
		req.DeliveryMethod = "対面・配送相談"
	}
	if req.ShippingDays <= 0 {
		req.ShippingDays = 2
	}
	result, err := r.DB.ExecContext(ctx,
		`UPDATE items SET title=?, description=?, category=?, condition_text=?, price_yen=?, image_url=?, delivery_method=?, shipping_days=?, ship_from_region=?, size=?, color=?, tags=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND seller_id=? AND status='available'`,
		req.Title, req.Description, req.Category, req.ConditionText, req.PriceYen, req.ImageURL, req.DeliveryMethod, req.ShippingDays, req.ShipFromRegion, req.Size, req.Color, req.Tags, itemID, sellerID)
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
	item, err := r.FindByID(ctx, itemID)
	if err != nil {
		return models.Item{}, err
	}
	// チェックリスト登録者へ変更通知を残します。
	rows, _ := r.DB.QueryContext(ctx, `SELECT user_id FROM checklist WHERE item_id = ? AND user_id <> ?`, itemID, sellerID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var uid int64
			if rows.Scan(&uid) == nil {
				_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, ?, ?)`, uid, itemID, "チェックリスト商品の更新", item.Title+" の商品情報が更新されました")
			}
		}
	}
	return item, nil
}

// Cancel は、出品者本人が販売中の商品を出品キャンセルへ変更する処理です。
// 商品を削除せずstatusをcanceledにすることで、出品履歴や通知から過去の操作を確認できるようにします。
func (r *ItemRepository) Cancel(ctx context.Context, itemID, sellerID int64) (models.Item, error) {
	// 出品キャンセルは、商品状態の変更と通知作成を一つのトランザクションで行います。
	// 途中で失敗した場合に「商品だけキャンセルされ、通知が残らない」という中途半端な状態を避けます。
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Item{}, err
	}
	defer tx.Rollback()

	// 通知文に商品名を入れるため、更新前に対象商品をロックして取得します。
	// FOR UPDATE により、同じ商品への同時キャンセルや購入処理との競合を防ぎます。
	var title string
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT title, status FROM items WHERE id=? AND seller_id=? FOR UPDATE`, itemID, sellerID).Scan(&title, &status); err != nil {
		return models.Item{}, err
	}
	if status != "available" {
		return models.Item{}, fmt.Errorf("商品が見つからないか、キャンセルできない状態です")
	}

	// 商品をキャンセル済みにします。履歴や通知から確認できるよう、DELETEではなくstatus更新にします。
	result, err := tx.ExecContext(ctx, `UPDATE items SET status='canceled', updated_at=CURRENT_TIMESTAMP WHERE id=? AND seller_id=? AND status='available'`, itemID, sellerID)
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

	// 出品者本人へキャンセル完了通知を作成します。
	if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '出品キャンセル完了', ?)`, sellerID, itemID, title+" の出品をキャンセルしました"); err != nil {
		return models.Item{}, err
	}

	// MySQLでは、同じトランザクション上で rows を開いたまま別のExecを行うと、
	// ドライバが「invalid connection」を返すことがあります。
	// そのため、まずチェックリスト登録者IDだけをすべて読み取り、rowsを閉じてから通知をINSERTします。
	rows, err := tx.QueryContext(ctx, `SELECT user_id FROM checklist WHERE item_id=? AND user_id<>?`, itemID, sellerID)
	if err != nil {
		return models.Item{}, err
	}
	checklistUserIDs := []int64{}
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			rows.Close()
			return models.Item{}, err
		}
		checklistUserIDs = append(checklistUserIDs, uid)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return models.Item{}, err
	}
	if err := rows.Close(); err != nil {
		return models.Item{}, err
	}

	// 対象商品をチェックリストに入れていたユーザーへ通知します。
	// 出品者本人は上で通知済みなので、重複しないようSQL側で除外済みです。
	for _, uid := range checklistUserIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, 'チェックリスト商品の出品キャンセル', ?)`, uid, itemID, title+" は出品者によりキャンセルされました"); err != nil {
			return models.Item{}, err
		}
	}

	// ここまで成功したらコミットします。
	if err := tx.Commit(); err != nil {
		return models.Item{}, err
	}

	// コミット後に通常のDB接続で商品を再取得します。
	// キャンセル済みの商品も出品履歴や通知から確認できるよう、FindByIDはstatusで除外しません。
	return r.FindByID(ctx, itemID)
}

// FindByID は、商品詳細・購入後の再取得・通知からの商品表示で使う単一商品取得です。
// キャンセル済みの商品も履歴として確認できるよう、ここではstatusによる除外を行いません。
func (r *ItemRepository) FindByID(ctx context.Context, id int64) (models.Item, error) {
	return scanItem(r.DB.QueryRowContext(ctx, itemSelect()+` WHERE i.id = ?`, id))
}

// Purchase は、購入者の残高引き落とし、商品ステータス変更、購入レコード作成をまとめて行います。
// 同時購入や残高の二重消費を防ぐため、商品行と購入者行をFOR UPDATEでロックしたトランザクション内で処理します。
func (r *ItemRepository) Purchase(ctx context.Context, itemID, buyerID int64, deliveryAddress string) (models.Purchase, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Purchase{}, err
	}
	defer tx.Rollback()
	var sellerID int64
	var priceYen, shippingDays, buyerBalance int
	var status string
	var title string
	err = tx.QueryRowContext(ctx, `SELECT seller_id, price_yen, status, shipping_days, title FROM items WHERE id=? FOR UPDATE`, itemID).Scan(&sellerID, &priceYen, &status, &shippingDays, &title)
	if err != nil {
		return models.Purchase{}, err
	}
	if sellerID == buyerID {
		return models.Purchase{}, fmt.Errorf("自分の商品は購入できません")
	}
	if status != "available" {
		return models.Purchase{}, fmt.Errorf("この商品は購入できません")
	}
	var blocked int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM blocked_users WHERE (blocker_id=? AND blocked_id=?) OR (blocker_id=? AND blocked_id=?)`, buyerID, sellerID, sellerID, buyerID).Scan(&blocked)
	if err != nil {
		return models.Purchase{}, err
	}
	if blocked > 0 {
		return models.Purchase{}, fmt.Errorf("ブロック関係にあるユーザーの商品は購入できません")
	}
	err = tx.QueryRowContext(ctx, `SELECT balance_coins FROM users WHERE id=? FOR UPDATE`, buyerID).Scan(&buyerBalance)
	if err != nil {
		return models.Purchase{}, err
	}
	if buyerBalance < priceYen {
		return models.Purchase{}, fmt.Errorf("残高不足です。チャージしてから購入手続きを行ってください")
	}
	if deliveryAddress == "" {
		_ = tx.QueryRowContext(ctx, `SELECT COALESCE(shipping_address, '') FROM users WHERE id=?`, buyerID).Scan(&deliveryAddress)
	}
	if strings.TrimSpace(deliveryAddress) == "" {
		return models.Purchase{}, fmt.Errorf("お届け先住所を入力してください")
	}
	// 購入手続き完了時点では、購入者の残高だけを差し引きます。
	// 出品者への入金は、商品到着後に購入者が受け取り評価を行ったCompleteで実行します。
	// これにより、フリマアプリで一般的な「一時預かり金（エスクロー）」に近い取引フローになります。
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance_coins=balance_coins-? WHERE id=?`, priceYen, buyerID); err != nil {
		return models.Purchase{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE items SET status='sold', updated_at=CURRENT_TIMESTAMP WHERE id=?`, itemID); err != nil {
		return models.Purchase{}, err
	}
	deadline := time.Now().AddDate(0, 0, shippingDays)
	result, err := tx.ExecContext(ctx, `INSERT INTO purchases (item_id, buyer_id, seller_id, price_yen, status, shipping_deadline, delivery_address) VALUES (?, ?, ?, ?, 'paid', ?, ?)`, itemID, buyerID, sellerID, priceYen, deadline, deliveryAddress)
	if err != nil {
		return models.Purchase{}, err
	}
	purchaseID, err := result.LastInsertId()
	if err != nil {
		return models.Purchase{}, err
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, ?, ?)`, sellerID, itemID, "商品が購入されました", title+" が購入されました。売上は購入者の受け取り評価後に残高へ反映されます。発送通知を行ってください")
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, ?, ?)`, buyerID, itemID, "購入手続きが完了しました", title+" の購入手続きが完了しました。残高から商品代金を差し引きました")
	if err := tx.Commit(); err != nil {
		return models.Purchase{}, err
	}
	return models.Purchase{ID: purchaseID, ItemID: itemID, BuyerID: buyerID, SellerID: sellerID, PriceYen: priceYen, Status: "paid", DeliveryAddress: deliveryAddress, ShippingDeadline: deadline}, nil
}

// Ship は、出品者が発送通知を押したときに購入ステータスを paid から shipped へ進めます。
// 購入者へは受け取り評価を促し、出品者へは発送通知済みであることを残します。
func (r *ItemRepository) Ship(ctx context.Context, itemID, sellerID int64) (models.Purchase, error) {
	var p models.Purchase
	res, err := r.DB.ExecContext(ctx, `UPDATE purchases SET status='shipped', shipped_at=CURRENT_TIMESTAMP WHERE item_id=? AND seller_id=? AND status='paid'`, itemID, sellerID)
	if err != nil {
		return p, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return p, fmt.Errorf("発送通知できる取引が見つかりません")
	}
	_ = r.DB.QueryRowContext(ctx, `SELECT id, item_id, buyer_id, seller_id, price_yen, status, delivery_address, created_at, shipping_deadline, shipped_at, completed_at FROM purchases WHERE item_id=?`, itemID).Scan(&p.ID, &p.ItemID, &p.BuyerID, &p.SellerID, &p.PriceYen, &p.Status, &p.DeliveryAddress, &p.CreatedAt, &p.ShippingDeadline, &p.ShippedAt, &p.CompletedAt)
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '発送通知', '出品者が発送通知を行いました。到着後に受け取り評価をしてください')`, p.BuyerID, itemID)
	_, _ = r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '発送通知送信済み', '発送通知を送信しました。購入者の受け取り評価をお待ちください')`, p.SellerID, itemID)
	return p, nil
}

// Complete は、購入者の受け取り評価によって取引を完了し、売上を出品者残高へ反映します。
// フリマのエスクロー風フローでは、購入時に預かった代金をこのタイミングで出品者へ移すのが重要です。
func (r *ItemRepository) Complete(ctx context.Context, itemID, buyerID int64, rating int, comment string) (models.Purchase, error) {
	if rating < 1 || rating > 5 {
		return models.Purchase{}, fmt.Errorf("評価は1〜5で入力してください")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Purchase{}, err
	}
	defer tx.Rollback()
	var p models.Purchase
	err = tx.QueryRowContext(ctx, `SELECT id,item_id,buyer_id,seller_id,price_yen,status,delivery_address,created_at,shipping_deadline,shipped_at,completed_at FROM purchases WHERE item_id=? AND buyer_id=? FOR UPDATE`, itemID, buyerID).Scan(&p.ID, &p.ItemID, &p.BuyerID, &p.SellerID, &p.PriceYen, &p.Status, &p.DeliveryAddress, &p.CreatedAt, &p.ShippingDeadline, &p.ShippedAt, &p.CompletedAt)
	if err != nil {
		return p, err
	}
	if p.Status != "shipped" {
		return p, fmt.Errorf("発送通知後に受け取り評価できます")
	}
	_, err = tx.ExecContext(ctx, `UPDATE purchases SET status='completed', completed_at=CURRENT_TIMESTAMP, rating=?, rating_comment=? WHERE id=?`, rating, comment, p.ID)
	if err != nil {
		return p, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE users SET rating_sum=rating_sum+?, rating_count=rating_count+1, transaction_count=transaction_count+1, balance_coins=balance_coins+?, sales_coins=sales_coins+? WHERE id=?`, rating, p.PriceYen, p.PriceYen, p.SellerID)
	if err != nil {
		return p, err
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '取引完了・売上反映', ?)`, p.SellerID, itemID, fmt.Sprintf("購入者が受け取り評価を行い、取引が完了しました。売上%sを残高へ反映しました", formatJPY(p.PriceYen)))
	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, '取引完了', '受け取り評価が完了しました')`, p.BuyerID, itemID)
	if err := tx.Commit(); err != nil {
		return p, err
	}
	return r.FindPurchaseByItem(ctx, itemID)
}

// FindPurchaseByItem は、商品IDから対応する購入レコードを取得します。
// 発送や完了後に最新のpurchase.statusや日時を返すため、各更新処理の最後で再取得に使います。
func (r *ItemRepository) FindPurchaseByItem(ctx context.Context, itemID int64) (models.Purchase, error) {
	var p models.Purchase
	err := r.DB.QueryRowContext(ctx, `SELECT id,item_id,buyer_id,seller_id,price_yen,status,delivery_address,created_at,shipping_deadline,shipped_at,completed_at FROM purchases WHERE item_id=?`, itemID).Scan(&p.ID, &p.ItemID, &p.BuyerID, &p.SellerID, &p.PriceYen, &p.Status, &p.DeliveryAddress, &p.CreatedAt, &p.ShippingDeadline, &p.ShippedAt, &p.CompletedAt)
	return p, err
}

// ListPurchasesByBuyer は、購入履歴ページに表示するための商品情報と購入情報をまとめて取得します。
// 取引状態、発送期限、評価コメントなどは purchases 側、商品名や画像は items 側からJOINして1つの履歴行にします。
func (r *ItemRepository) ListPurchasesByBuyer(ctx context.Context, buyerID int64) ([]models.PurchaseHistory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT p.id, i.id, i.product_code, i.seller_id, u.name, CASE WHEN u.rating_count=0 THEN 0 ELSE u.rating_sum/u.rating_count END, u.rating_count, i.title, i.description, i.category, i.condition_text, p.price_yen, COALESCE(i.image_url,''), i.status, p.status, i.delivery_method, i.shipping_days, i.ship_from_region, p.delivery_address, p.created_at, p.shipping_deadline, p.shipped_at, p.completed_at, p.rating, COALESCE(p.rating_comment,'') FROM purchases p JOIN items i ON i.id=p.item_id JOIN users u ON u.id=i.seller_id WHERE p.buyer_id=? ORDER BY p.created_at DESC`, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PurchaseHistory
	for rows.Next() {
		var x models.PurchaseHistory
		var rating sql.NullInt64
		if err := rows.Scan(&x.PurchaseID, &x.ItemID, &x.ProductCode, &x.SellerID, &x.SellerName, &x.SellerRatingAverage, &x.SellerRatingCount, &x.Title, &x.Description, &x.Category, &x.ConditionText, &x.PriceYen, &x.ImageURL, &x.Status, &x.PurchaseStatus, &x.DeliveryMethod, &x.ShippingDays, &x.ShipFromRegion, &x.DeliveryAddress, &x.PurchasedAt, &x.ShippingDeadline, &x.ShippedAt, &x.CompletedAt, &rating, &x.RatingComment); err != nil {
			return nil, err
		}
		if rating.Valid {
			v := int(rating.Int64)
			x.Rating = &v
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ListChecklist は、ユーザーが「気になる」に入れた商品を新しい順で取得します。
// キャンセル済み商品は購入対象として表示しにくいため、一覧からは除外しています。
func (r *ItemRepository) ListChecklist(ctx context.Context, userID int64) ([]models.Item, error) {
	// checklist に該当する商品が0件でも JSON では [] として扱いやすいよう、
	// nil ではなく空スライスで初期化してから append する。
	items := []models.Item{}

	rows, err := r.DB.QueryContext(ctx, itemSelect()+` JOIN checklist c2 ON c2.item_id=i.id WHERE c2.user_id=? AND i.status <> 'canceled' ORDER BY c2.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// IsInChecklist は、商品詳細画面のハート状態を判定するための存在確認です。
// レコードがなければエラーではなく false を返し、画面側が通常の未登録状態として扱えるようにします。
func (r *ItemRepository) IsInChecklist(ctx context.Context, userID, itemID int64) (bool, error) {
	var exists int
	err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM checklist WHERE user_id=? AND item_id=?`, userID, itemID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// AddChecklist は、商品をユーザーのチェックリストへ追加します。
// INSERT IGNORE により、同じ商品を連打しても重複エラーにせず、結果的に登録済み状態へ収束させます。
func (r *ItemRepository) AddChecklist(ctx context.Context, userID, itemID int64) error {
	_, err := r.DB.ExecContext(ctx, `INSERT IGNORE INTO checklist (user_id,item_id,last_seen_updated_at) SELECT ?, id, updated_at FROM items WHERE id=?`, userID, itemID)
	return err
}

// RemoveChecklist は、商品をユーザーのチェックリストから外します。
// 存在しない組み合わせを削除しても問題ないため、画面の再クリックや通信リトライにも比較的強い操作です。
func (r *ItemRepository) RemoveChecklist(ctx context.Context, userID, itemID int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM checklist WHERE user_id=? AND item_id=?`, userID, itemID)
	return err
}

// Recommend は、ログインユーザー向けの簡易おすすめ商品を返します。
// 現在はチェックリスト数と更新日時を使った軽量な推薦で、本人の出品とブロック関係の商品は除外します。
func (r *ItemRepository) Recommend(ctx context.Context, userID int64) ([]models.Item, error) {
	items := []models.Item{}
	rows, err := r.DB.QueryContext(ctx, itemSelect()+` WHERE i.status='available' AND i.seller_id<>? AND NOT EXISTS (SELECT 1 FROM blocked_users b WHERE (b.blocker_id=? AND b.blocked_id=i.seller_id) OR (b.blocker_id=i.seller_id AND b.blocked_id=?)) ORDER BY checklist_count DESC, i.updated_at DESC LIMIT 8`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SimilarPriceStats は、同カテゴリ商品の価格分布を取得します。
// AI価格アドバイスでは、外部データや重いML推論が使えない場合でも、
// 現在アプリ内に存在する類似出品の中央値・件数を用いて価格感を説明します。
func (r *ItemRepository) SimilarPriceStats(ctx context.Context, category string, excludeID int64) (count int, min int, max int, avg float64, err error) {
	row := r.DB.QueryRowContext(ctx, `
        SELECT COUNT(*), COALESCE(MIN(price_yen),0), COALESCE(MAX(price_yen),0), COALESCE(AVG(price_yen),0)
        FROM items
        WHERE status <> 'canceled' AND category = ? AND id <> ?`, category, excludeID)
	err = row.Scan(&count, &min, &max, &avg)
	return
}

// CreateStaleListingAdviceNotifications は、一定期間売れ残っている出品に対し、
// MerRec由来のカテゴリ別観点とアプリ内の成約価格傾向を使って、出品者へ改善提案通知を作成します。
//
// 本番ではCloud SchedulerやCloud Tasksで定期実行するのが自然ですが、
// ハッカソンのローカル・Cloud Runデモではサーバ起動時と簡易tickerで動く方が確認しやすいため、
// main.goからこの関数を定期的に呼びます。
func (r *ItemRepository) CreateStaleListingAdviceNotifications(ctx context.Context, days int) (int, error) {
	if days <= 0 {
		days = 7
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := r.DB.QueryContext(ctx, `
		SELECT i.id,
		       i.seller_id,
		       i.title,
		       i.category,
		       i.price_yen,
		       COALESCE(i.size, ''),
		       COALESCE(i.tags, ''),
		       i.updated_at,
		       COALESCE((
		         SELECT AVG(p.price_yen)
		         FROM purchases p
		         JOIN items sold_item ON sold_item.id = p.item_id
		         WHERE p.status = 'completed'
		           AND sold_item.category = i.category
		       ), 0) AS completed_avg_price
		FROM items i
		WHERE i.status = 'available'
		  AND i.updated_at <= ?
		  AND NOT EXISTS (
		    SELECT 1
		    FROM notifications n
		    WHERE n.user_id = i.seller_id
		      AND n.item_id = i.id
		      AND n.title = 'AI販売改善提案'
		      AND n.created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)
		  )
		ORDER BY i.updated_at ASC
		LIMIT 50`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	created := 0
	for rows.Next() {
		var itemID, sellerID int64
		var title, category, size, tags string
		var priceYen int
		var updatedAt time.Time
		var completedAvgPrice float64
		if err := rows.Scan(&itemID, &sellerID, &title, &category, &priceYen, &size, &tags, &updatedAt, &completedAvgPrice); err != nil {
			return created, err
		}
		body := buildStaleListingAdviceBody(title, category, priceYen, size, tags, completedAvgPrice)
		if _, err := r.DB.ExecContext(ctx, `INSERT INTO notifications (user_id, item_id, title, body) VALUES (?, ?, 'AI販売改善提案', ?)`, sellerID, itemID, body); err != nil {
			return created, err
		}
		created++
	}
	return created, rows.Err()
}

// buildStaleListingAdviceBody は、売れ残り通知の本文を商品情報から組み立てます。
// MerRecのようなC2C行動分析で重要になりやすい「情報不足」「検索されやすさ」「価格感」を説明可能な文章に変換します。
func buildStaleListingAdviceBody(title, category string, priceYen int, size, tags string, completedAvgPrice float64) string {
	// MerRecのようなC2Cデータでよく効く「購入前の不安解消」「検索語の補強」「価格調整」を、
	// 商品カテゴリ・現在価格・成約平均価格から説明可能な形で通知します。
	advice := []string{}
	trimmedSize := strings.TrimSpace(size)
	trimmedTags := strings.TrimSpace(tags)
	if trimmedSize == "" {
		advice = append(advice, "サイズ・型番・実寸など、購入判断に必要な具体情報を追記すると安心感が上がります。")
	}
	if trimmedTags == "" || len(strings.Split(trimmedTags, ",")) < 3 {
		advice = append(advice, "検索用タグを3〜5個程度追加すると、自然言語検索やカテゴリ検索で見つかりやすくなります。")
	}
	for _, hint := range staleCategoryHints(category) {
		advice = append(advice, hint)
		break
	}
	if completedAvgPrice > 0 && float64(priceYen) > completedAvgPrice*1.1 {
		advice = append(advice, fmt.Sprintf("同カテゴリの成約平均が約%d円のため、%d〜%d円程度への調整を検討できます。", int(completedAvgPrice), int(completedAvgPrice*0.95), int(completedAvgPrice*1.05)))
	} else {
		advice = append(advice, "すぐに値下げしない場合でも、写真順を見直し、1枚目に商品の状態が最も伝わる画像を置くと効果的です。")
	}
	return fmt.Sprintf("「%s」は7日以上Availableのままです。MerRec風の過去取引分析では、%s", strings.TrimSpace(title), strings.Join(advice, " "))
}

// staleCategoryHints は、カテゴリごとに購入者が不安に感じやすい確認項目を返します。
// AIが使えない環境でも、最低限カテゴリに合った改善提案を出せるようにするフォールバック知識です。
func staleCategoryHints(category string) []string {
	c := strings.TrimSpace(category)
	switch {
	case strings.Contains(c, "本") || strings.Contains(c, "教材"):
		return []string{"版・年度、書き込みの有無、解答冊子の有無を明記すると購入前の不安が下がります。"}
	case strings.Contains(c, "スマホ") || strings.Contains(c, "PC") || strings.Contains(c, "家電"):
		return []string{"動作確認日、対応端子、付属品、バッテリー状態を追記すると比較されやすくなります。"}
	case strings.Contains(c, "ファッション"):
		return []string{"着用回数、実寸、色味、汚れの位置を追記するとサイズ不安を減らせます。"}
	case strings.Contains(c, "家具") || strings.Contains(c, "インテリア"):
		return []string{"縦横高さ、設置イメージ、部屋での色味が分かる写真を足すと反応が上がりやすいです。"}
	default:
		return []string{"状態が伝わる写真、付属品、受け渡し条件を追記すると購入判断がしやすくなります。"}
	}
}

// BuildFilterFromQuery は、HTTPのURLクエリをItemFilterへ変換します。
// Handler層でSQL条件を直接組まず、Repositoryが理解できる検索条件の構造体へ一度まとめるための境界関数です。
func BuildFilterFromQuery(values map[string][]string) ItemFilter {
	get := func(k string) string {
		if len(values[k]) == 0 {
			return ""
		}
		return strings.TrimSpace(values[k][0])
	}
	atoi := func(s string) int { var v int; fmt.Sscanf(s, "%d", &v); return v }
	return ItemFilter{Query: get("q"), Category: get("category"), Size: get("size"), Color: get("color"), ConditionText: get("condition"), Status: get("status"), MinPrice: atoi(get("minPrice")), MaxPrice: atoi(get("maxPrice")), Tag: get("tag"), Sort: get("sort"), DeliveryWithin: get("deliveryWithin")}
}
