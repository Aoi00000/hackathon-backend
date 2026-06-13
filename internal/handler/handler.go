package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"hackathon-backend/internal/ai"
	"hackathon-backend/internal/auth"
	"hackathon-backend/internal/config"
	"hackathon-backend/internal/httpx"
	"hackathon-backend/internal/models"
	"hackathon-backend/internal/repository"
)

// Handler はHTTPハンドラが必要とする依存関係をまとめた構造体です。
// DB操作、AI呼び出し、設定を一箇所に集めることで、各メソッドの引数を単純にしています。
type Handler struct {
	Config   config.Config
	Users    repository.UserRepository
	Items    repository.ItemRepository
	Messages repository.MessageRepository
	AI       *ai.Client
}

// New はHandlerを初期化します。
func New(cfg config.Config, database *sql.DB) *Handler {
	return &Handler{
		Config:   cfg,
		Users:    repository.UserRepository{DB: database},
		Items:    repository.ItemRepository{DB: database},
		Messages: repository.MessageRepository{DB: database},
		AI:       ai.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel),
	}
}

// Register はユーザー登録APIです。
// POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "名前、メールアドレス、8文字以上のパスワードを入力してください")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "パスワード処理に失敗しました")
		return
	}

	user, err := h.Users.Create(r.Context(), req.Name, req.Email, string(passwordHash))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "このメールアドレスは既に使われている可能性があります")
		return
	}

	token, err := auth.GenerateToken(user.ID, h.Config.JWTSecret)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "トークン生成に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

// Login はログインAPIです。
// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	user, err := h.Users.FindByEmail(r.Context(), strings.TrimSpace(req.Email))
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "メールアドレスまたはパスワードが正しくありません")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "メールアドレスまたはパスワードが正しくありません")
		return
	}

	token, err := auth.GenerateToken(user.ID, h.Config.JWTSecret)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "トークン生成に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}

// Me はログイン中ユーザーを返すAPIです。
// GET /api/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	user, err := h.Users.FindByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "ユーザーが見つかりません")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user)
}

// ListItems は商品一覧APIです。
// GET /api/items?q=keyword
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	items, err := h.Items.List(r.Context(), q)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品一覧の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, items)
}

// ListMyItems はログイン中ユーザーの出品履歴を返します。
// GET /api/me/items
func (h *Handler) ListMyItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	items, err := h.Items.ListBySeller(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "出品履歴の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, items)
}

// CreateItem は商品出品APIです。
// POST /api/items
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	var req models.CreateItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	trimCreateItemRequest(&req)
	if req.Title == "" || req.Description == "" || req.Category == "" || req.ConditionText == "" || req.PriceYen <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "商品名、説明、カテゴリ、状態、1円以上の価格を入力してください")
		return
	}

	item, err := h.Items.Create(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品の作成に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, item)
}

// UpdateItem は出品者本人が商品情報を編集するAPIです。
// PUT /api/items/{id}
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	itemID, ok := parseIDFromPath(r.URL.Path, "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	var req models.UpdateItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.ConditionText = strings.TrimSpace(req.ConditionText)
	req.ImageURL = strings.TrimSpace(req.ImageURL)

	if req.Title == "" || req.Description == "" || req.Category == "" || req.ConditionText == "" || req.PriceYen <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "商品名、説明、カテゴリ、状態、1円以上の価格を入力してください")
		return
	}

	item, err := h.Items.Update(r.Context(), itemID, userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

// CancelItem は出品者本人が出品をキャンセルするAPIです。
// POST /api/items/{id}/cancel
func (h *Handler) CancelItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/cancel"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	item, err := h.Items.Cancel(r.Context(), itemID, userID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

// GetItem は商品詳細APIです。
// GET /api/items/{id}
func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseIDFromPath(r.URL.Path, "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	item, err := h.Items.FindByID(r.Context(), itemID)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "商品が見つかりません")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

// PurchaseItem は商品購入APIです。
// POST /api/items/{id}/purchase
func (h *Handler) PurchaseItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/purchase"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	purchase, err := h.Items.Purchase(r.Context(), itemID, userID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, purchase)
}

// ListPurchaseHistory はログイン中ユーザーの購入履歴を返します。
// GET /api/me/purchases
func (h *Handler) ListPurchaseHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	history, err := h.Items.ListPurchasesByBuyer(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "購入履歴の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, history)
}

// ListChecklist はログイン中ユーザーのチェックリストを返します。
// GET /api/me/checklist
func (h *Handler) ListChecklist(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	items, err := h.Items.ListChecklist(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "チェックリストの取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, items)
}

// GetChecklistStatus は指定商品がチェックリストに入っているかを返します。
// GET /api/items/{id}/checklist
func (h *Handler) GetChecklistStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/checklist"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	checked, err := h.Items.IsInChecklist(r.Context(), userID, itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "チェックリスト状態の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ChecklistStatus{Checked: checked})
}

// AddChecklist は商品をチェックリストに追加します。
// POST /api/items/{id}/checklist
func (h *Handler) AddChecklist(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/checklist"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	item, err := h.Items.FindByID(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "商品が見つかりません")
		return
	}
	if item.SellerID == userID {
		httpx.WriteError(w, http.StatusBadRequest, "自分の商品はチェックリストに追加できません")
		return
	}

	if err := h.Items.AddChecklist(r.Context(), userID, itemID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "チェックリストへの追加に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ChecklistStatus{Checked: true})
}

// RemoveChecklist は商品をチェックリストから削除します。
// DELETE /api/items/{id}/checklist
func (h *Handler) RemoveChecklist(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/checklist"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	if err := h.Items.RemoveChecklist(r.Context(), userID, itemID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "チェックリストからの削除に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.ChecklistStatus{Checked: false})
}

// GenerateDescription はGeminiで商品説明を生成するAPIです。
// POST /api/ai/generate-description
func (h *Handler) GenerateDescription(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateDescriptionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(req.Category)
	req.ConditionText = strings.TrimSpace(req.ConditionText)
	req.Keywords = strings.TrimSpace(req.Keywords)

	if req.Title == "" || req.Category == "" || req.ConditionText == "" {
		httpx.WriteError(w, http.StatusBadRequest, "AI生成には商品名、カテゴリ、状態が必要です")
		return
	}

	prompt := ai.BuildDescriptionPrompt(req.Title, req.Category, req.ConditionText, req.Keywords)
	text, err := h.AI.GenerateText(prompt)
	if err != nil {
		log.Printf("gemini generate description failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる説明生成に失敗しました: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text})
}

// AskItem はGeminiで商品について質問するAPIです。
// POST /api/items/{id}/ask
func (h *Handler) AskItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/ask"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	var req models.AskItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		httpx.WriteError(w, http.StatusBadRequest, "質問を入力してください")
		return
	}

	item, err := h.Items.FindByID(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "商品が見つかりません")
		return
	}

	prompt := ai.BuildItemQAPrompt(item.Title, item.Description, item.Category, item.ConditionText, req.Question)
	text, err := h.AI.GenerateText(prompt)
	if err != nil {
		log.Printf("gemini item qa failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる回答生成に失敗しました: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text})
}

// ListMessages は商品に紐づくコメント一覧APIです。
// GET /api/items/{id}/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/messages"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	messages, err := h.Messages.ListByItem(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "コメント一覧の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, messages)
}

// CreateMessage はコメント送信APIです。
// POST /api/items/{id}/messages
func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}

	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/messages"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	var req models.CreateMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	body := strings.TrimSpace(req.Body)
	if body == "" {
		httpx.WriteError(w, http.StatusBadRequest, "コメント本文を入力してください")
		return
	}

	msg, err := h.Messages.Create(r.Context(), itemID, userID, req.ParentMessageID, body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, msg)
}

// trimCreateItemRequest は出品リクエスト内の文字列前後の空白を取り除きます。
func trimCreateItemRequest(req *models.CreateItemRequest) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.ConditionText = strings.TrimSpace(req.ConditionText)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
}

// parseIDFromPath は /api/items/123 のようなURLからIDを取り出す補助関数です。
// 本格的なルーターを導入しなくても、標準ライブラリだけでMVPを組めるようにしています。
func parseIDFromPath(path string, prefix string) (int64, bool) {
	raw := strings.TrimPrefix(path, prefix)
	if raw == path || raw == "" {
		return 0, false
	}
	if strings.Contains(raw, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
