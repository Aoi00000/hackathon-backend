// ============================================================
// ファイル概要: hackathon-backend/internal/models/models.go
// 役割: DB行、APIレスポンス、AIリクエスト/レスポンスで共有する構造体を集約します。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
// Package models は、DBとAPIレスポンスで共有するデータ構造を定義します。
//
// JSONタグはフロントエンドのTypeScript型と対応しているため、名前を変える場合は src/types.ts も合わせて確認してください。
package models

import "time"

// User は users テーブルの1行に対応する構造体です。
// APIレスポンスでは password_hash を返さないよう、PasswordHashにはjsonタグを付けていません。
// balanceCoins はアプリ内仮想通貨の利用可能残高、salesCoins は売上金です。
// 【詳細コメント】User は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type User struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】Name は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Name string `json:"name"`
	// 【構造体フィールド】Email は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Email string `json:"email"`
	// 【構造体フィールド】PasswordHash は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PasswordHash string `json:"-"`
	// 【構造体フィールド】BalanceCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BalanceCoins int `json:"balanceCoins"`
	// 【構造体フィールド】SalesCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SalesCoins int `json:"salesCoins"`
	// 【構造体フィールド】RatingAverage は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	RatingAverage float64 `json:"ratingAverage"`
	// 【構造体フィールド】RatingCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	RatingCount int `json:"ratingCount"`
	// 【構造体フィールド】TransactionCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	TransactionCount int `json:"transactionCount"`
	// 【構造体フィールド】ShippingRegion は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingRegion string `json:"shippingRegion"`
	// 【構造体フィールド】ShippingAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingAddress string `json:"shippingAddress"`
	// 【構造体フィールド】MonthlySpendCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	MonthlySpendCoins int `json:"monthlySpendCoins"`
	// 【構造体フィールド】TotalSpendCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	TotalSpendCoins int `json:"totalSpendCoins"`
	// 【構造体フィールド】MonthlySalesCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	MonthlySalesCoins int `json:"monthlySalesCoins"`
	// 【構造体フィールド】TotalSalesCoins は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	TotalSalesCoins int `json:"totalSalesCoins"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// Item は items テーブルの1行に対応する構造体です。
// ProductCode は画面表示用の独自商品IDです。DBの内部IDとは別に見せることで、同名商品の識別を容易にします。
// 【詳細コメント】Item は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type Item struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】ProductCode は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ProductCode string `json:"productCode"`
	// 【構造体フィールド】SellerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerID int64 `json:"sellerId"`
	// 【構造体フィールド】SellerName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerName string `json:"sellerName"`
	// 【構造体フィールド】SellerRatingAverage は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerRatingAverage float64 `json:"sellerRatingAverage"`
	// 【構造体フィールド】SellerRatingCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerRatingCount int `json:"sellerRatingCount"`
	// 【構造体フィールド】SellerTransactionCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerTransactionCount int `json:"sellerTransactionCount"`
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】Description は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Description string `json:"description"`
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category"`
	// 【構造体フィールド】ConditionText は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ConditionText string `json:"conditionText"`
	// 【構造体フィールド】PriceYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PriceYen int `json:"priceYen"`
	// 【構造体フィールド】ImageURL は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ImageURL string `json:"imageUrl"`
	// 【構造体フィールド】Status は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Status string `json:"status"`
	// 【構造体フィールド】DeliveryMethod は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryMethod string `json:"deliveryMethod"`
	// 【構造体フィールド】ShippingDays は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDays int `json:"shippingDays"`
	// 【構造体フィールド】ShipFromRegion は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShipFromRegion string `json:"shipFromRegion"`
	// 【構造体フィールド】Size は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Size string `json:"size"`
	// 【構造体フィールド】Color は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Color string `json:"color"`
	// 【構造体フィールド】Tags は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Tags string `json:"tags"`
	// 【構造体フィールド】ChecklistCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ChecklistCount int `json:"checklistCount"`
	// 【構造体フィールド】BuyerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BuyerID *int64 `json:"buyerId,omitempty"`
	// 【構造体フィールド】BuyerName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BuyerName string `json:"buyerName,omitempty"`
	// 【構造体フィールド】BuyerShippingAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BuyerShippingAddress string `json:"buyerShippingAddress,omitempty"`
	// 【構造体フィールド】PurchaseID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchaseID *int64 `json:"purchaseId,omitempty"`
	// 【構造体フィールド】PurchaseStatus は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchaseStatus string `json:"purchaseStatus,omitempty"`
	// 【構造体フィールド】PurchaseCreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchaseCreatedAt *time.Time `json:"purchaseCreatedAt,omitempty"`
	// 【構造体フィールド】ShippingDeadline は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDeadline *time.Time `json:"shippingDeadline,omitempty"`
	// 【構造体フィールド】ShippedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippedAt *time.Time `json:"shippedAt,omitempty"`
	// 【構造体フィールド】CompletedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
	// 【構造体フィールド】UpdatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UpdatedAt time.Time `json:"updatedAt"`
}

// Purchase は purchases テーブルの1行に対応する構造体です。
// 【詳細コメント】Purchase は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type Purchase struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】ItemID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ItemID int64 `json:"itemId"`
	// 【構造体フィールド】BuyerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BuyerID int64 `json:"buyerId"`
	// 【構造体フィールド】SellerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerID int64 `json:"sellerId"`
	// 【構造体フィールド】PriceYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PriceYen int `json:"priceYen"`
	// 【構造体フィールド】Status は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Status string `json:"status"`
	// 【構造体フィールド】DeliveryAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryAddress string `json:"deliveryAddress"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
	// 【構造体フィールド】ShippingDeadline は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDeadline time.Time `json:"shippingDeadline"`
	// 【構造体フィールド】ShippedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippedAt *time.Time `json:"shippedAt,omitempty"`
	// 【構造体フィールド】CompletedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// PurchaseHistory は購入履歴画面で表示しやすいよう、購入情報と商品情報をまとめた構造体です。
// 【詳細コメント】PurchaseHistory は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type PurchaseHistory struct {
	// 【構造体フィールド】PurchaseID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchaseID int64 `json:"purchaseId"`
	// 【構造体フィールド】ItemID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ItemID int64 `json:"itemId"`
	// 【構造体フィールド】ProductCode は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ProductCode string `json:"productCode"`
	// 【構造体フィールド】SellerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerID int64 `json:"sellerId"`
	// 【構造体フィールド】SellerName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerName string `json:"sellerName"`
	// 【構造体フィールド】SellerRatingAverage は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerRatingAverage float64 `json:"sellerRatingAverage"`
	// 【構造体フィールド】SellerRatingCount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SellerRatingCount int `json:"sellerRatingCount"`
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】Description は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Description string `json:"description"`
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category"`
	// 【構造体フィールド】ConditionText は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ConditionText string `json:"conditionText"`
	// 【構造体フィールド】PriceYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PriceYen int `json:"priceYen"`
	// 【構造体フィールド】ImageURL は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ImageURL string `json:"imageUrl"`
	// 【構造体フィールド】Status は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Status string `json:"status"`
	// 【構造体フィールド】PurchaseStatus は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchaseStatus string `json:"purchaseStatus"`
	// 【構造体フィールド】DeliveryMethod は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryMethod string `json:"deliveryMethod"`
	// 【構造体フィールド】ShippingDays は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDays int `json:"shippingDays"`
	// 【構造体フィールド】ShipFromRegion は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShipFromRegion string `json:"shipFromRegion"`
	// 【構造体フィールド】DeliveryAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryAddress string `json:"deliveryAddress"`
	// 【構造体フィールド】PurchasedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PurchasedAt time.Time `json:"purchasedAt"`
	// 【構造体フィールド】ShippingDeadline は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDeadline time.Time `json:"shippingDeadline"`
	// 【構造体フィールド】ShippedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippedAt *time.Time `json:"shippedAt,omitempty"`
	// 【構造体フィールド】CompletedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// 【構造体フィールド】Rating は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Rating *int `json:"rating,omitempty"`
	// 【構造体フィールド】RatingComment は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	RatingComment string `json:"ratingComment,omitempty"`
}

// Message は公開コメント欄の1投稿に対応する構造体です。
// ParentMessageID が nil のものは親コメント、値が入っているものは返信です。
// 【詳細コメント】Message は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type Message struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】ItemID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ItemID int64 `json:"itemId"`
	// 【構造体フィールド】ParentMessageID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
	// 【構造体フィールド】SenderID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SenderID int64 `json:"senderId"`
	// 【構造体フィールド】SenderName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SenderName string `json:"senderName"`
	// 【構造体フィールド】ReceiverID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverID int64 `json:"receiverId"`
	// 【構造体フィールド】ReceiverName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverName string `json:"receiverName"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】IsSeller は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	IsSeller bool `json:"isSeller"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
	// 【構造体フィールド】UpdatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UpdatedAt time.Time `json:"updatedAt"`
}

// PrivateMessage は購入検討者と出品者だけが見られる非公開DMです。
// 【詳細コメント】PrivateMessage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type PrivateMessage struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】ItemID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ItemID int64 `json:"itemId"`
	// 【構造体フィールド】ParentPrivateMessageID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ParentPrivateMessageID *int64 `json:"parentMessageId,omitempty"`
	// 【構造体フィールド】SenderID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SenderID int64 `json:"senderId"`
	// 【構造体フィールド】SenderName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SenderName string `json:"senderName"`
	// 【構造体フィールド】ReceiverID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverID int64 `json:"receiverId"`
	// 【構造体フィールド】ReceiverName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverName string `json:"receiverName"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// Notification はユーザーへの簡易通知です。
// 【詳細コメント】Notification は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type Notification struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
	// 【構造体フィールド】ItemID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ItemID *int64 `json:"itemId,omitempty"`
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】ReadAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReadAt *time.Time `json:"readAt,omitempty"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// SavedSearch は保存した検索条件です。
// 【詳細コメント】SavedSearch は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type SavedSearch struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
	// 【構造体フィールド】Name は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Name string `json:"name"`
	// 【構造体フィールド】QueryJSON は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	QueryJSON string `json:"queryJson"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// BlockedUser はブロック済みユーザーです。
// 【詳細コメント】BlockedUser は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type BlockedUser struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】BlockerID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BlockerID int64 `json:"blockerId"`
	// 【構造体フィールド】BlockedID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BlockedID int64 `json:"blockedId"`
	// 【構造体フィールド】BlockedName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	BlockedName string `json:"blockedName"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// SupportMessage はユーザーから運営への問い合わせです。
// 【詳細コメント】SupportMessage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type SupportMessage struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
	// 【構造体フィールド】UserName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserName string `json:"userName"`
	// 【構造体フィールド】Subject は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Subject string `json:"subject"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// RecommendationResponse はAIおすすめ欄で使うレスポンスです。
// 【詳細コメント】RecommendationResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type RecommendationResponse struct {
	// 【構造体フィールド】Reason は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Reason string `json:"reason"`
	// 【構造体フィールド】Items は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Items []Item `json:"items"`
}

// RegisterRequest はユーザー登録APIのリクエストJSONです。
// 【詳細コメント】RegisterRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type RegisterRequest struct {
	// 【構造体フィールド】Name は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Name string `json:"name"`
	// 【構造体フィールド】Email は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Email string `json:"email"`
	// 【構造体フィールド】Password は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Password string `json:"password"`
}

// 【詳細コメント】LoginRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type LoginRequest struct {
	// 【構造体フィールド】Email は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Email string `json:"email"`
	// 【構造体フィールド】Password は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Password string `json:"password"`
}

// 【詳細コメント】AuthResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AuthResponse struct {
	// 【構造体フィールド】Token は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Token string `json:"token"`
	// 【構造体フィールド】User は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	User User `json:"user"`
}

// 【詳細コメント】CreateItemRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CreateItemRequest struct {
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】Description は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Description string `json:"description"`
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category"`
	// 【構造体フィールド】ConditionText は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ConditionText string `json:"conditionText"`
	// 【構造体フィールド】PriceYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PriceYen int `json:"priceYen"`
	// 【構造体フィールド】ImageURL は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ImageURL string `json:"imageUrl"`
	// 【構造体フィールド】DeliveryMethod は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryMethod string `json:"deliveryMethod"`
	// 【構造体フィールド】ShippingDays は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingDays int `json:"shippingDays"`
	// 【構造体フィールド】ShipFromRegion は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShipFromRegion string `json:"shipFromRegion"`
	// 【構造体フィールド】Size は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Size string `json:"size"`
	// 【構造体フィールド】Color は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Color string `json:"color"`
	// 【構造体フィールド】Tags は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Tags string `json:"tags"`
}

// 【詳細コメント】UpdateItemRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type UpdateItemRequest = CreateItemRequest

// 【詳細コメント】PurchaseRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type PurchaseRequest struct {
	// 【構造体フィールド】DeliveryAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryAddress string `json:"deliveryAddress"`
}

// 【詳細コメント】CompletePurchaseRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CompletePurchaseRequest struct {
	// 【構造体フィールド】Rating は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Rating int `json:"rating"`
	// 【構造体フィールド】RatingComment は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	RatingComment string `json:"ratingComment"`
}

// 【詳細コメント】ChargeRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type ChargeRequest struct {
	// 【構造体フィールド】Amount は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Amount int `json:"amount"`
}

// 【詳細コメント】UpdateProfileRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type UpdateProfileRequest struct {
	// 【構造体フィールド】ShippingRegion は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingRegion string `json:"shippingRegion"`
	// 【構造体フィールド】ShippingAddress は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ShippingAddress string `json:"shippingAddress"`
}

// 【詳細コメント】GenerateDescriptionRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type GenerateDescriptionRequest struct {
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category"`
	// 【構造体フィールド】ConditionText は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ConditionText string `json:"conditionText"`
	// 【構造体フィールド】Keywords は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Keywords string `json:"keywords"`
}

// NaturalSearchRequest は、商品一覧トップの自然言語検索で使うリクエストです。
// 例: 「予算1万円以内で、使用感が少ない参考書を安い順に探して」
// 【詳細コメント】NaturalSearchRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type NaturalSearchRequest struct {
	// 【構造体フィールド】Query は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Query string `json:"query"`
}

// NaturalSearchResponse は、自然言語を既存の商品検索パラメータへ変換した結果です。
// フロントエンドはこの値をそのまま商品一覧の検索フォーム状態へ反映します。
// 【詳細コメント】NaturalSearchResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type NaturalSearchResponse struct {
	// 【構造体フィールド】Q は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Q string `json:"q,omitempty"`
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category,omitempty"`
	// 【構造体フィールド】Size は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Size string `json:"size,omitempty"`
	// 【構造体フィールド】Color は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Color string `json:"color,omitempty"`
	// 【構造体フィールド】Condition は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Condition string `json:"condition,omitempty"`
	// 【構造体フィールド】Status は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Status string `json:"status,omitempty"`
	// 【構造体フィールド】MinPrice は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	MinPrice string `json:"minPrice,omitempty"`
	// 【構造体フィールド】MaxPrice は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	MaxPrice string `json:"maxPrice,omitempty"`
	// 【構造体フィールド】Tag は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Tag string `json:"tag,omitempty"`
	// 【構造体フィールド】DeliveryWithin は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DeliveryWithin string `json:"deliveryWithin,omitempty"`
	// 【構造体フィールド】Sort は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Sort string `json:"sort,omitempty"`
	// 【構造体フィールド】Explanation は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Explanation string `json:"explanation,omitempty"`
	// 【構造体フィールド】Notice は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Notice string `json:"notice,omitempty"`
	// 【構造体フィールド】UsedFallback は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UsedFallback bool `json:"usedFallback,omitempty"`
}

// 【詳細コメント】AskItemRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AskItemRequest struct {
	// 【構造体フィールド】Question は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Question string `json:"question"`
}

// PriceNegotiationRequest は、商品詳細の価格交渉アシスタントで使う入力です。
// DesiredPriceYen は、購入検討者が希望する金額、または出品者が検討したい金額です。
// 【詳細コメント】PriceNegotiationRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type PriceNegotiationRequest struct {
	// 【構造体フィールド】DesiredPriceYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	DesiredPriceYen int `json:"desiredPriceYen"`
}

// 【詳細コメント】AITextResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AITextResponse struct {
	// 【構造体フィールド】Text は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Text string `json:"text"`
	// 【構造体フィールド】Notice は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Notice string `json:"notice,omitempty"`
	// 【構造体フィールド】UsedFallback は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UsedFallback bool `json:"usedFallback,omitempty"`
}

// ItemAIAnalysisResponse は、商品詳細で購入前の不安点・質問候補・カテゴリ不整合・価格感を返すレスポンスです。
// Geminiが利用できない場合でも、バックエンドのルールベース解析で最低限の結果を返します。
// 【詳細コメント】ItemAIAnalysisResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type ItemAIAnalysisResponse struct {
	// 【構造体フィールド】RiskPoints は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	RiskPoints []string `json:"riskPoints"`
	// 【構造体フィールド】SuggestedQuestions は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SuggestedQuestions []string `json:"suggestedQuestions"`
	// 【構造体フィールド】Inconsistencies は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Inconsistencies []string `json:"inconsistencies"`
	// 【構造体フィールド】PriceInsight は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	PriceInsight string `json:"priceInsight"`
	// 【構造体フィールド】CategoryReviewHints は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CategoryReviewHints []string `json:"categoryReviewHints"`
}

// CategoryKnowledgeResponse は、MerRec などのC2Cレコメンド知識から、
// そのカテゴリで購入者が気にしやすい点を出品画面へ返すレスポンスです。
// 【詳細コメント】CategoryKnowledgeResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CategoryKnowledgeResponse struct {
	// 【構造体フィールド】Category は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Category string `json:"category"`
	// 【構造体フィールド】Tips は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Tips []string `json:"tips"`
}

// 【詳細コメント】CreateMessageRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CreateMessageRequest struct {
	// 【構造体フィールド】ParentMessageID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】ReceiverID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverID int64 `json:"receiverId,omitempty"`
}

// 【詳細コメント】CreatePrivateMessageRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CreatePrivateMessageRequest struct {
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】ReceiverID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ReceiverID int64 `json:"receiverId,omitempty"`
	// 【構造体フィールド】ParentMessageID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
}

// 【詳細コメント】ChecklistStatus は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type ChecklistStatus struct {
	// 【構造体フィールド】Checked は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Checked bool `json:"checked"`
}

// 【詳細コメント】SaveSearchRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type SaveSearchRequest struct {
	// 【構造体フィールド】Name は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Name string `json:"name"`
	// 【構造体フィールド】QueryJSON は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	QueryJSON string `json:"queryJson"`
}

// 【詳細コメント】BlockUserRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type BlockUserRequest struct {
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
}

// 【詳細コメント】SupportMessageRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type SupportMessageRequest struct {
	// 【構造体フィールド】Subject は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Subject string `json:"subject"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
}

// MonthlyMoneySummary はマイページの月別収支グラフに使う集計行です。
// 【詳細コメント】MonthlyMoneySummary は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type MonthlyMoneySummary struct {
	// 【構造体フィールド】Month は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Month string `json:"month"`
	// 【構造体フィールド】SalesYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SalesYen int `json:"salesYen"`
	// 【構造体フィールド】SpendYen は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SpendYen int `json:"spendYen"`
}

// PaymentMethod はユーザーがマイページで登録する支払い方法です。
// カード番号とセキュリティコードはデモ用途として保存しますが、APIレスポンスには下4桁と表示名だけ返します。
// 実運用ではカード情報を直接保存せず、決済代行サービスのトークンだけを保存してください。
// 【詳細コメント】PaymentMethod は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type PaymentMethod struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
	// 【構造体フィールド】Label は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Label string `json:"label"`
	// 【構造体フィールド】CardLast4 は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CardLast4 string `json:"cardLast4"`
	// 【構造体フィールド】HolderName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	HolderName string `json:"holderName"`
	// 【構造体フィールド】ExpiryMonth は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ExpiryMonth int `json:"expiryMonth"`
	// 【構造体フィールド】ExpiryYear は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ExpiryYear int `json:"expiryYear"`
	// 【構造体フィールド】IsDefault は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	IsDefault bool `json:"isDefault"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// 【詳細コメント】CreatePaymentMethodRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CreatePaymentMethodRequest struct {
	// 【構造体フィールド】Label は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Label string `json:"label"`
	// 【構造体フィールド】CardNumber は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CardNumber string `json:"cardNumber"`
	// 【構造体フィールド】HolderName は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	HolderName string `json:"holderName"`
	// 【構造体フィールド】ExpiryMonth は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ExpiryMonth int `json:"expiryMonth"`
	// 【構造体フィールド】ExpiryYear は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ExpiryYear int `json:"expiryYear"`
	// 【構造体フィールド】SecurityCode は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	SecurityCode string `json:"securityCode"`
	// 【構造体フィールド】IsDefault は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	IsDefault bool `json:"isDefault"`
}

// AIChatThread は、AI対話ページで話題ごとに分けて保存する会話スレッドです。
// 1つのスレッドが「模様替え相談」「勉強グッズ相談」のような1つの話題に対応します。
// 【詳細コメント】AIChatThread は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AIChatThread struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】UserID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserID int64 `json:"userId"`
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
	// 【構造体フィールド】UpdatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIChatMessage は、AI対話スレッド内の1発言です。
// Role は user または assistant に限定し、UI側で吹き出しの左右・色を切り替えます。
// 【詳細コメント】AIChatMessage は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AIChatMessage struct {
	// 【構造体フィールド】ID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ID int64 `json:"id"`
	// 【構造体フィールド】ThreadID は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	ThreadID int64 `json:"threadId"`
	// 【構造体フィールド】Role は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Role string `json:"role"`
	// 【構造体フィールド】Body は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Body string `json:"body"`
	// 【構造体フィールド】Notice は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Notice string `json:"notice,omitempty"`
	// 【構造体フィールド】UsedFallback は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UsedFallback bool `json:"usedFallback"`
	// 【構造体フィールド】CreatedAt は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	CreatedAt time.Time `json:"createdAt"`
}

// CreateAIChatThreadRequest は、ユーザーがAI対話ページで新しい話題を作るときの入力です。
// Title を空にした場合は、バックエンド側で「新しい相談」として補完します。
// 【詳細コメント】CreateAIChatThreadRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type CreateAIChatThreadRequest struct {
	// 【構造体フィールド】Title は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Title string `json:"title"`
}

// AIChatTurnRequest は、既存スレッドへユーザー発言を1件追加するときの入力です。
// 【詳細コメント】AIChatTurnRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AIChatTurnRequest struct {
	// 【構造体フィールド】Message は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Message string `json:"message"`
}

// AIChatTurnResponse は、ユーザー発言とAI返信をまとめて返すレスポンスです。
// フロントエンドは、この2件を既存のメッセージ配列へappendするだけで画面を更新できます。
// 【詳細コメント】AIChatTurnResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AIChatTurnResponse struct {
	// 【構造体フィールド】Thread は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Thread AIChatThread `json:"thread"`
	// 【構造体フィールド】UserMessage は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	UserMessage AIChatMessage `json:"userMessage"`
	// 【構造体フィールド】AssistantMessage は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	AssistantMessage AIChatMessage `json:"assistantMessage"`
}

// 【詳細コメント】AIChatRequest は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type AIChatRequest struct {
	// 【構造体フィールド】Message は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Message string `json:"message"`
}

// 【詳細コメント】ErrorResponse は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
type ErrorResponse struct {
	// 【構造体フィールド】Error は、DB列またはAPI JSONの1項目に対応します。omitemptyやjsonタグが画面表示・更新処理に影響します。
	Error string `json:"error"`
}
