// ============================================================
// ファイル概要: hackathon-backend/internal/models/models.go
// 役割: DB行、APIレスポンス、AIリクエスト/レスポンスで共有する構造体を集約します。
//
// ============================================================
// 実装詳細メモ:
// DB行、APIリクエスト、APIレスポンス、AI応答をGo構造体として定義します。
// jsonタグはフロントエンドsrc/types.tsと対応するため、フィールド追加時は両方の契約を合わせます。
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

// RegisterRequest/LoginRequest/AuthResponse は認証APIの入出力です。
// Register/Loginはどちらも成功時にJWTとUserを返し、フロントエンドのAuthContextがそのまま保存します。
// Passwordはリクエストでだけ使い、UserレスポンスにはPasswordHashを出さない設計です。
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest はログインフォームから送られるメールアドレスとパスワードです。
// HandlerでFindByEmailとパスワード照合を行い、成功時にJWTを発行します。
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse は認証成功時に返すJWTとユーザー情報です。
// フロントエンドはtokenをsessionStorageへ保存し、userをヘッダーやマイページ初期表示に使います。
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateItemRequest は出品・編集フォームから送る商品入力です。
// ImageURLは旧名ですが、現在は複数画像/動画をJSON文字列化したメディア配列も受け取ります。
// DeliveryMethod/ShippingDays/ShipFromRegionは購入前の不安を減らすため、説明文だけでなく構造化項目として保持します。
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

// UpdateItemRequest は、商品編集時の入力をCreateItemRequestと同じ形で扱うための別名です。
// 出品と編集で入力項目が同じなので、型を共有してAPI契約の重複を避けます。
type UpdateItemRequest = CreateItemRequest

// PurchaseRequest/CompletePurchaseRequest は取引状態を進める入力です。
// 購入時点のDeliveryAddressはユーザーのプロフィール変更後も取引履歴に残すため、purchasesへ保存します。
type PurchaseRequest struct {
	DeliveryAddress string `json:"deliveryAddress"`
}

// CompletePurchaseRequest は、受け取り評価時に購入者が送る評価点とコメントです。
// Complete処理ではこの評価が出品者のrating_sum/rating_countへ反映されます。
type CompletePurchaseRequest struct {
	Rating        int    `json:"rating"`
	RatingComment string `json:"ratingComment"`
}

// ChargeRequest は、マイページや購入手続き画面から残高チャージする金額です。
// チャージ処理前に既定の支払い方法が登録されているかをRepositoryで確認します。
type ChargeRequest struct {
	Amount int `json:"amount"`
}

// UpdateProfileRequest は購入時の配送先初期値と、出品時の発送元地域の初期値に使います。
type UpdateProfileRequest struct {
	ShippingRegion  string `json:"shippingRegion"`
	ShippingAddress string `json:"shippingAddress"`
}

// GenerateDescriptionRequest は出品画面のAI商品説明生成で使います。
// Keywordsには出品者メモや注意点を入れ、AIが商品状態を誇張せず説明できるようにします。
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

// AskItemRequest は商品詳細で購入検討者がAIへ質問するときの入力です。
// AIは商品説明に書かれている情報だけを根拠にし、不明点は出品者確認へ誘導します。
type AskItemRequest struct {
	Question string `json:"question"`
}

// PriceNegotiationRequest は、商品詳細の価格交渉アシスタントで使う入力です。
// DesiredPriceYen は、購入検討者が希望する金額、または出品者が検討したい金額です。
type PriceNegotiationRequest struct {
	DesiredPriceYen int `json:"desiredPriceYen"`
}

// AITextResponse は、AI生成系APIの共通レスポンスです。
// Textに生成本文、Noticeに外部AI失敗時などの補足、UsedFallbackにローカル生成利用の有無を入れます。
type AITextResponse struct {
	Text         string `json:"text"`
	Notice       string `json:"notice,omitempty"`
	UsedFallback bool   `json:"usedFallback,omitempty"`
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

// CreateMessageRequest/CreatePrivateMessageRequest は公開コメントと非公開DMの投稿入力です。
// ParentMessageIDがある場合は返信、ReceiverIDがある場合は通知やDM相手を明示します。
// 公開コメントでは出品者宛が基本、非公開DMでは購入検討者と出品者の1対1会話を想定します。
type CreateMessageRequest struct {
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
	Body            string `json:"body"`
	ReceiverID      int64  `json:"receiverId,omitempty"`
}

// CreatePrivateMessageRequest は、商品詳細の非公開DM送信で使う入力です。
// ReceiverIDを省略した場合はRepository側で出品者または返信相手を補完します。
type CreatePrivateMessageRequest struct {
	Body            string `json:"body"`
	ReceiverID      int64  `json:"receiverId,omitempty"`
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
}

// ChecklistStatus は、商品がログインユーザーのチェックリストに入っているかを返します。
// 商品詳細ページのハートボタン表示と追加/削除後の状態更新に使います。
type ChecklistStatus struct {
	Checked bool `json:"checked"`
}

// SaveSearchRequestは商品一覧の検索条件をJSON文字列として保存する入力です。
// 新しいフィルタが増えてもqueryJsonの中身だけ増やせるため、DBスキーマ変更を避けられます。
type SaveSearchRequest struct {
	Name      string `json:"name"`
	QueryJSON string `json:"queryJson"`
}

// BlockUserRequestはコメント・DM・商品一覧表示から相手ユーザーを避けるための入力です。
type BlockUserRequest struct {
	UserID int64 `json:"userId"`
}

// SupportMessageRequestはマイページから運営へ問い合わせを送る入力です。
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

// CreatePaymentMethodRequest は、マイページの支払い方法登録フォームから送られる入力です。
// デモ用途ではカード番号とセキュリティコードを受けますが、レスポンスには下4桁だけを返します。
type CreatePaymentMethodRequest struct {
	Label        string `json:"label"`
	CardNumber   string `json:"cardNumber"`
	HolderName   string `json:"holderName"`
	ExpiryMonth  int    `json:"expiryMonth"`
	ExpiryYear   int    `json:"expiryYear"`
	SecurityCode string `json:"securityCode"`
	IsDefault    bool   `json:"isDefault"`
}

// AIChatThread は、AI対話ページで話題ごとに分けて保存する会話スレッドです。
// 1つのスレッドが「模様替え相談」「勉強グッズ相談」のような1つの話題に対応します。
type AIChatThread struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AIChatMessage は、AI対話スレッド内の1発言です。
// Role は user または assistant に限定し、UI側で吹き出しの左右・色を切り替えます。
type AIChatMessage struct {
	ID           int64     `json:"id"`
	ThreadID     int64     `json:"threadId"`
	Role         string    `json:"role"`
	Body         string    `json:"body"`
	Notice       string    `json:"notice,omitempty"`
	UsedFallback bool      `json:"usedFallback"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CreateAIChatThreadRequest は、ユーザーがAI対話ページで新しい話題を作るときの入力です。
// Title を空にした場合は、バックエンド側で「新しい相談」として補完します。
type CreateAIChatThreadRequest struct {
	Title string `json:"title"`
}

// AIChatTurnRequest は、既存スレッドへユーザー発言を1件追加するときの入力です。
type AIChatTurnRequest struct {
	Message string `json:"message"`
}

// AIChatTurnResponse は、ユーザー発言とAI返信をまとめて返すレスポンスです。
// フロントエンドは、この2件を既存のメッセージ配列へappendするだけで画面を更新できます。
type AIChatTurnResponse struct {
	Thread           AIChatThread  `json:"thread"`
	UserMessage      AIChatMessage `json:"userMessage"`
	AssistantMessage AIChatMessage `json:"assistantMessage"`
}

// AIChatRequest は、古い単発AIチャットAPIとの互換用入力です。
// 現在のAI対話ページでは、履歴保存できるAIChatTurnRequest/Responseを主に使います。
type AIChatRequest struct {
	Message string `json:"message"`
}

// ErrorResponse は、APIエラーをJSONで返すときの共通形です。
// httpx.WriteErrorがこの形式を使い、フロントエンドのrequest関数がerror文字列を取り出します。
type ErrorResponse struct {
	Error string `json:"error"`
}
