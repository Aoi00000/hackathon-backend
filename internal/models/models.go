package models

import "time"

// User は users テーブルの1行に対応する構造体です。
// APIレスポンスでは password_hash を返さないよう、PasswordHashにはjsonタグを付けていません。
type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Item は items テーブルの1行に対応する構造体です。
// Statusは available / sold / canceled のいずれかを想定します。
type Item struct {
	ID            int64     `json:"id"`
	SellerID      int64     `json:"sellerId"`
	SellerName    string    `json:"sellerName"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	ConditionText string    `json:"conditionText"`
	PriceYen      int       `json:"priceYen"`
	ImageURL      string    `json:"imageUrl"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Purchase は purchases テーブルの1行に対応する構造体です。
type Purchase struct {
	ID        int64     `json:"id"`
	ItemID    int64     `json:"itemId"`
	BuyerID   int64     `json:"buyerId"`
	SellerID  int64     `json:"sellerId"`
	PriceYen  int       `json:"priceYen"`
	CreatedAt time.Time `json:"createdAt"`
}

// PurchaseHistory は購入履歴画面で表示しやすいよう、購入情報と商品情報をまとめた構造体です。
type PurchaseHistory struct {
	PurchaseID    int64     `json:"purchaseId"`
	ItemID        int64     `json:"itemId"`
	SellerID      int64     `json:"sellerId"`
	SellerName    string    `json:"sellerName"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	ConditionText string    `json:"conditionText"`
	PriceYen      int       `json:"priceYen"`
	ImageURL      string    `json:"imageUrl"`
	Status        string    `json:"status"`
	PurchasedAt   time.Time `json:"purchasedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Message は商品コメント欄の1投稿に対応する構造体です。
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

// RegisterRequest はユーザー登録APIのリクエストJSONです。
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest はログインAPIのリクエストJSONです。
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse はログインや登録成功時に返すJSONです。
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateItemRequest は商品出品APIのリクエストJSONです。
type CreateItemRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	ConditionText string `json:"conditionText"`
	PriceYen      int    `json:"priceYen"`
	ImageURL      string `json:"imageUrl"`
}

// UpdateItemRequest は商品情報編集APIのリクエストJSONです。
// 出品者だけが、自分の商品を編集するために使います。
type UpdateItemRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	ConditionText string `json:"conditionText"`
	PriceYen      int    `json:"priceYen"`
	ImageURL      string `json:"imageUrl"`
}

// GenerateDescriptionRequest はGeminiで商品説明を生成するAPIの入力です。
type GenerateDescriptionRequest struct {
	Title         string `json:"title"`
	Category      string `json:"category"`
	ConditionText string `json:"conditionText"`
	Keywords      string `json:"keywords"`
}

// AskItemRequest は商品についてGeminiに質問するAPIの入力です。
type AskItemRequest struct {
	Question string `json:"question"`
}

// AITextResponse はAI生成テキストを返す共通レスポンスです。
type AITextResponse struct {
	Text string `json:"text"`
}

// CreateMessageRequest はコメント投稿APIのリクエストJSONです。
// ParentMessageID を指定すると、既存コメントへの返信になります。
type CreateMessageRequest struct {
	ParentMessageID *int64 `json:"parentMessageId,omitempty"`
	Body            string `json:"body"`
	ReceiverID      int64  `json:"receiverId,omitempty"` // 旧実装との互換用。新実装ではサーバ側で送信先を決めます。
}

// ChecklistStatus は商品がチェックリストに入っているかを返すレスポンスです。
type ChecklistStatus struct {
	Checked bool `json:"checked"`
}

// ErrorResponse はエラー時に返すJSONの形です。
type ErrorResponse struct {
	Error string `json:"error"`
}
