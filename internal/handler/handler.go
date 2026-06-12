package handler

import (
	"database/sql"
	"errors"
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
		Config: cfg,
		Users: repository.UserRepository{
			DB: database,
		},
		Items: repository.ItemRepository{
			DB: database,
		},
		Messages: repository.MessageRepository{
			DB: database,
		},
		AI: ai.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel),
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

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.ConditionText = strings.TrimSpace(req.ConditionText)

	if req.Title == "" || req.Description == "" || req.Category == "" || req.ConditionText == "" || req.PriceYen <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "商品名、説明、カテゴリ、状態、価格を入力してください")
		return
	}

	item, err := h.Items.Create(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品の作成に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, item)
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

// GenerateDescription はGeminiで商品説明を生成するAPIです。
// POST /api/ai/generate-description
func (h *Handler) GenerateDescription(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateDescriptionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}

	prompt := ai.BuildDescriptionPrompt(req.Title, req.Category, req.ConditionText, req.Keywords)

	text, err := h.AI.GenerateText(prompt)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる説明生成に失敗しました")
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

	item, err := h.Items.FindByID(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "商品が見つかりません")
		return
	}

	prompt := ai.BuildItemQAPrompt(item.Title, item.Description, item.Category, item.ConditionText, req.Question)

	text, err := h.AI.GenerateText(prompt)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる回答生成に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text})
}

// ListMessages は商品に紐づくDM一覧APIです。
// GET /api/items/{id}/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/messages"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}

	messages, err := h.Messages.ListByItem(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "メッセージ一覧の取得に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, messages)
}

// CreateMessage はDM送信APIです。
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

	if req.ReceiverID <= 0 || strings.TrimSpace(req.Body) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "送信先と本文を入力してください")
		return
	}

	msg, err := h.Messages.Create(r.Context(), itemID, userID, req.ReceiverID, strings.TrimSpace(req.Body))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "メッセージ送信に失敗しました")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, msg)
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
