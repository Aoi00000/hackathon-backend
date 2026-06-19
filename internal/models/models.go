// Package models は、DBとAPIレスポンスで共有するデータ構造を定義します。
//
// JSONタグはフロントエンドのTypeScript型と対応しているため、名前を変える場合は src/types.ts も合わせて確認してください。
package models

import "time"

// User は users テーブルの1行に対応する構造体です。
// APIレスポンスでは password_hash を返さないよう、PasswordHashにはjsonタグを付けていません。
// balanceCoins はアプリ内仮想通貨の利用可能残高、salesCoins は売上金です。
type User struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"-"`
	BalanceCoins      int       `json:"balanceCoins"`
	SalesCoins        int       `json:"salesCoins"`
	RatingAverage     float64   `json:"ratingAverage"`
	RatingCount       int       `json:"ratingCount"`
	TransactionCount  int       `json:"transactionCount"`
	ShippingRegion    string    `json:"shippingRegion"`
	ShippingAddress   string    `json:"shippingAddress"`
	MonthlySpendCoins int       `json:"monthlySpendCoins"`
	TotalSpendCoins   int       `json:"totalSpendCoins"`
	MonthlySalesCoins int       `json:"monthlySalesCoins"`
	TotalSalesCoins   int       `json:"totalSalesCoins"`
	CreatedAt         time.Time `json:"createdAt"`
}

// Item は items テーブルの1行に対応する構造体です。
// ProductCode は画面表示用の独自商品IDです。DBの内部IDとは別に見せることで、同名商品の識別を容易にします。
type Item struct {
	ID                     int64      `json:"id"`
	ProductCode            string     `json:"productCode"`
	SellerID               int64      `json:"sellerId"`
	SellerName             string     `json:"sellerName"`
	SellerRatingAverage    float64    `json:"sellerRatingAverage"`
	SellerRatingCount      int        `json:"sellerRatingCount"`
	SellerTransactionCount int        `json:"sellerTransactionCount"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	Category               string     `json:"category"`
	ConditionText          string     `json:"conditionText"`
	PriceYen               int        `json:"priceYen"`
	ImageURL               string     `json:"imageUrl"`
	Status                 string     `json:"status"`
	DeliveryMethod         string     `json:"deliveryMethod"`
	ShippingDays           int        `json:"shippingDays"`
	ShipFromRegion         string     `json:"shipFromRegion"`
	Size                   string     `json:"size"`
	Color                  string     `json:"color"`
	Tags                   string     `json:"tags"`
	ChecklistCount         int        `json:"checklistCount"`
	BuyerID                *int64     `json:"buyerId,omitempty"`
	BuyerName              string     `json:"buyerName,omitempty"`
	BuyerShippingAddress   string     `json:"buyerShippingAddress,omitempty"`
	PurchaseID             *int64     `json:"purchaseId,omitempty"`
	PurchaseStatus         string     `json:"purchaseStatus,omitempty"`
	PurchaseCreatedAt      *time.Time `json:"purchaseCreatedAt,omitempty"`
	ShippingDeadline       *time.Time `json:"shippingDeadline,omitempty"`
	ShippedAt              *time.Time `json:"shippedAt,omitempty"`
	CompletedAt            *time.Time `json:"completedAt,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

// Purchase は purchases テーブルの1行に対応する構造体です。
type Purchase struct {
	ID               int64      `json:"id"`
	ItemID           int64      `json:"itemId"`
	BuyerID          int64      `json:"buyerId"`
	SellerID         int64      `json:"sellerId"`
	PriceYen         int        `json:"priceYen"`
	Status           string     `json:"status"`
	DeliveryAddress  string     `json:"deliveryAddress"`
	CreatedAt        time.Time  `json:"createdAt"`
	ShippingDeadline time.Time  `json:"shippingDeadline"`
	ShippedAt        *time.Time `json:"shippedAt,omitempty"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// PurchaseHistory は購入履歴画面で表示しやすいよう、購入情報と商品情報をまとめた構造体です。
type PurchaseHistory struct {
	PurchaseID          int64      `json:"purchaseId"`
	ItemID              int64      `json:"itemId"`
	ProductCode         string     `json:"productCode"`
	SellerID            int64      `json:"sellerId"`
	SellerName          string     `json:"sellerName"`
	SellerRatingAverage float64    `json:"sellerRatingAverage"`
	SellerRatingCount   int        `json:"sellerRatingCount"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Category            string     `json:"category"`
	ConditionText       string     `json:"conditionText"`
	PriceYen            int        `json:"priceYen"`
	ImageURL            string     `json:"imageUrl"`
	Status              string     `json:"status"`
	PurchaseStatus      string     `json:"purchaseStatus"`
	DeliveryMethod      string     `json:"deliveryMethod"`
	ShippingDays        int        `json:"shippingDays"`
	ShipFromRegion      string     `json:"shipFromRegion"`
	DeliveryAddress     string     `json:"deliveryAddress"`
	PurchasedAt         time.Time  `json:"purchasedAt"`
	ShippingDeadline    time.Time  `json:"shippingDeadline"`
	ShippedAt           *time.Time `json:"shippedAt,omitempty"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
	Rating              *int       `json:"rating,omitempty"`
	RatingComment       string     `json:"ratingComment,omitempty"`
}

// Message は公開コメント欄の1投稿に対応する構造体です。
// ParentMessageID が nil のものは親コメント、値が入っているものは返信です。
type Message struct {
	ID              int64     `json:"id"`
	ItemID          int64     `json:"itemId"`
	ParentMessageID *int64    `json:"parentMessageId,omitempty"`
	SenderID        int64     `json:"senderId"`
	SenderName      string    `json:"senderName"`
	ReceiverID      int64     `json:"receiverId"`
	ReceiverName    string    `json:"receiverName"`
	Body            string    `json:"body"`
	IsSeller        bool      `json:"isSeller"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// PrivateMessage は購入検討者と出品者だけが見られる非公開DMです。
type PrivateMessage struct {
	ID                     int64     `json:"id"`
	ItemID                 int64     `json:"itemId"`
	ParentPrivateMessageID *int64    `json:"parentMessageId,omitempty"`
	SenderID               int64     `json:"senderId"`
	SenderName             string    `json:"senderName"`
	ReceiverID             int64     `json:"receiverId"`
	ReceiverName           string    `json:"receiverName"`
	Body                   string    `json:"body"`
	CreatedAt              time.Time `json:"createdAt"`
}

// Notification はユーザーへの簡易通知です。
type Notification struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"userId"`
	ItemID    *int64     `json:"itemId,omitempty"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// SavedSearch は保存した検索条件です。
type SavedSearch struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Name      string    `json:"name"`
	QueryJSON string    `json:"queryJson"`
	CreatedAt time.Time `json:"createdAt"`
}

// BlockedUser はブロック済みユーザーです。
type BlockedUser struct {
	ID          int64     `json:"id"`
	BlockerID   int64     `json:"blockerId"`
	BlockedID   int64     `json:"blockedId"`
	BlockedName string    `json:"blockedName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SupportMessage はユーザーから運営への問い合わせです。
type SupportMessage struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	UserName  string    `json:"userName"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// RecommendationResponse はAIおすすめ欄で使うレスポンスです。
type RecommendationResponse struct {
	Reason string `json:"reason"`
	Items  []Item `json:"items"`
}

// RegisterRequest はユーザー登録APIのリクエストJSONです。
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateItemRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	ConditionText  string `json:"conditionText"`
	PriceYen       int    `json:"priceYen"`
	ImageURL       string `json:"imageUrl"`
	DeliveryMethod string `json:"deliveryMethod"`
	ShippingDays   int    `json:"shippingDays"`
	ShipFromRegion string `json:"shipFromRegion"`
	Size           string `json:"size"`
	Color          string `json:"color"`
	Tags           string `json:"tags"`
}

type UpdateItemRequest = CreateItemRequest

type PurchaseRequest struct {
	DeliveryAddress string `json:"deliveryAddress"`
}

type CompletePurchaseRequest struct {
	Rating        int    `json:"rating"`
	RatingComment string `json:"ratingComment"`
}

type ChargeRequest struct {
	Amount int `json:"amount"`
}

type UpdateProfileRequest struct {
	ShippingRegion  string `json:"shippingRegion"`
	ShippingAddress string `json:"shippingAddress"`
}

type GenerateDescriptionRequest struct {
	Title         string `json:"title"`
	Category      string `json:"category"`
	ConditionText string `json:"conditionText"`
	Keywords      string `json:"keywords"`
}

// NaturalSearchRequest は、商品一覧トップの自然言語検索で使うリクエストです。
// 例: 「予算1万円以内で、使用感が少ない参考書を安い順に探して」
type NaturalSearchRequest struct {
	Query string `json:"query"`
}

// NaturalSearchResponse は、自然言語を既存の商品検索パラメータへ変換した結果です。
// フロントエンドはこの値をそのまま商品一覧の検索フォーム状態へ反映します。
type NaturalSearchResponse struct {
	Q              string `json:"q,omitempty"`
	Category       string `json:"category,omitempty"`
	Size           string `json:"size,omitempty"`
	Color          string `json:"color,omitempty"`
	Condition      string `json:"condition,omitempty"`
	Status         string `json:"status,omitempty"`
	MinPrice       string `json:"minPrice,omitempty"`
	MaxPrice       string `json:"maxPrice,omitempty"`
	Tag            string `json:"tag,omitempty"`
	DeliveryWithin string `json:"deliveryWithin,omitempty"`
	Sort           string `json:"sort,omitempty"`
	Explanation    string `json:"explanation,omitempty"`
	Notice         string `json:"notice,omitempty"`
	UsedFallback   bool   `json:"usedFallback,omitempty"`
}

type AskItemRequest struct {
	Question string `json:"question"`
}

type AITextResponse struct {
	Text         string `json:"text"`
	Notice       string `json:"notice,omitempty"`
	UsedFallback bool   `json:"usedFallback,omitempty"`
}

// AITranslateRequest は、UIの英語表示切り替えでユーザー入力テキストを英訳するためのリクエストです。
// 日本語に戻すときは再翻訳せず、フロントエンド側で元の日本語テキストを表示します。
type AITranslateRequest struct {
	Text string `json:"text"`
}

// ItemAIAnalysisResponse は、商品詳細で購入前の不安点・質問候補・カテゴリ不整合・価格感を返すレスポンスです。
// Geminiが利用できない場合でも、バックエンドのルールベース解析で最低限の結果を返します。
type ItemAIAnalysisResponse struct {
	RiskPoints          []string `json:"riskPoints"`
	SuggestedQuestions  []string `json:"suggestedQuestions"`
	Inconsistencies     []string `json:"inconsistencies"`
	PriceInsight        string   `json:"priceInsight"`
	CategoryReviewHints []string `json:"categoryReviewHints"`
}

// CategoryKnowledgeResponse は、MerRec などのC2Cレコメンド知識から、
// そのカテゴリで購入者が気にしやすい点を出品画面へ返すレスポンスです。
type CategoryKnowledgeResponse struct {
	Category string   `json:"category"`
	Tips     []string `json:"tips"`
}

type CreateMessageRequest struct {
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
	Body            string `json:"body"`
	ReceiverID      int64  `json:"receiverId,omitempty"`
}

type CreatePrivateMessageRequest struct {
	Body            string `json:"body"`
	ReceiverID      int64  `json:"receiverId,omitempty"`
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
}

type ChecklistStatus struct {
	Checked bool `json:"checked"`
}

type SaveSearchRequest struct {
	Name      string `json:"name"`
	QueryJSON string `json:"queryJson"`
}

type BlockUserRequest struct {
	UserID int64 `json:"userId"`
}

type SupportMessageRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// MonthlyMoneySummary はマイページの月別収支グラフに使う集計行です。
type MonthlyMoneySummary struct {
	Month    string `json:"month"`
	SalesYen int    `json:"salesYen"`
	SpendYen int    `json:"spendYen"`
}

// PaymentMethod はユーザーがマイページで登録する支払い方法です。
// カード番号とセキュリティコードはデモ用途として保存しますが、APIレスポンスには下4桁と表示名だけ返します。
// 実運用ではカード情報を直接保存せず、決済代行サービスのトークンだけを保存してください。
type PaymentMethod struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	Label       string    `json:"label"`
	CardLast4   string    `json:"cardLast4"`
	HolderName  string    `json:"holderName"`
	ExpiryMonth int       `json:"expiryMonth"`
	ExpiryYear  int       `json:"expiryYear"`
	IsDefault   bool      `json:"isDefault"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreatePaymentMethodRequest struct {
	Label        string `json:"label"`
	CardNumber   string `json:"cardNumber"`
	HolderName   string `json:"holderName"`
	ExpiryMonth  int    `json:"expiryMonth"`
	ExpiryYear   int    `json:"expiryYear"`
	SecurityCode string `json:"securityCode"`
	IsDefault    bool   `json:"isDefault"`
}

type AIChatRequest struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
