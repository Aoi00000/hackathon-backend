package handler

import (
	"database/sql"
	"errors"
	"fmt"
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

type Handler struct {
	Config   config.Config
	Users    repository.UserRepository
	Items    repository.ItemRepository
	Messages repository.MessageRepository
	AI       *ai.Client
}

func New(cfg config.Config, database *sql.DB) *Handler {
	return &Handler{
		Config:   cfg,
		Users:    repository.UserRepository{DB: database},
		Items:    repository.ItemRepository{DB: database},
		Messages: repository.MessageRepository{DB: database},
		AI:       ai.NewClient(cfg.AIProvider, cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GoogleProjectID, cfg.VertexLocation),
	}
}

func (h *Handler) optionalUserID(r *http.Request) *int64 {
	id, err := auth.UserIDFromRequest(r, h.Config.JWTSecret)
	if err != nil {
		return nil
	}
	return &id
}

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
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "パスワード処理に失敗しました")
		return
	}
	user, err := h.Users.Create(r.Context(), req.Name, req.Email, string(hash))
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

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.UpdateProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	req.ShippingRegion = strings.TrimSpace(req.ShippingRegion)
	req.ShippingAddress = strings.TrimSpace(req.ShippingAddress)
	if req.ShippingRegion == "" || req.ShippingAddress == "" {
		httpx.WriteError(w, http.StatusBadRequest, "発送元地域とお届け先住所は必須です")
		return
	}
	user, err := h.Users.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "プロフィール更新に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) Charge(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.ChargeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	user, err := h.Users.Charge(r.Context(), userID, req.Amount)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	filter := repository.BuildFilterFromQuery(r.URL.Query())
	items, err := h.Items.List(r.Context(), filter, h.optionalUserID(r))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品一覧の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

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

func trimItemRequest(req *models.CreateItemRequest) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.ConditionText = strings.TrimSpace(req.ConditionText)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	req.DeliveryMethod = strings.TrimSpace(req.DeliveryMethod)
	req.ShipFromRegion = strings.TrimSpace(req.ShipFromRegion)
	req.Size = strings.TrimSpace(req.Size)
	req.Color = strings.TrimSpace(req.Color)
	req.Tags = strings.TrimSpace(req.Tags)
}

func validateItemRequest(req models.CreateItemRequest) error {
	if req.Title == "" || req.Description == "" || req.Category == "" || req.ConditionText == "" || req.PriceYen <= 0 {
		return fmt.Errorf("商品名、説明、カテゴリ、状態、1円以上の価格を入力してください")
	}
	if req.ShipFromRegion == "" {
		return fmt.Errorf("発送元の地域を入力してください")
	}
	if req.DeliveryMethod == "" {
		return fmt.Errorf("商品の受け渡し方法を選択してください")
	}
	if req.ShippingDays <= 0 {
		return fmt.Errorf("発送までの日数は1日以上で入力してください")
	}
	return nil
}

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
	trimItemRequest(&req)
	if err := validateItemRequest(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.Items.Create(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "商品の作成に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, item)
}

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
	var req models.CreateItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	trimItemRequest(&req)
	if err := validateItemRequest(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.Items.Update(r.Context(), itemID, userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}

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
	// 出品一覧ではブロック相手の商品を非表示にしますが、
	// 購入履歴・出品履歴・通知からの遷移では、取引確認のため商品詳細を表示できるようにします。
	httpx.WriteJSON(w, http.StatusOK, item)
}

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
	var req models.PurchaseRequest
	_ = httpx.DecodeJSON(r, &req)
	purchase, err := h.Items.Purchase(r.Context(), itemID, userID, strings.TrimSpace(req.DeliveryAddress))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, purchase)
}

func (h *Handler) ShipItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/ship"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}
	p, err := h.Items.Ship(r.Context(), itemID, userID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) CompleteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/complete"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}
	var req models.CompletePurchaseRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	p, err := h.Items.Complete(r.Context(), itemID, userID, req.Rating, req.RatingComment)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

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
	text, err := h.AI.GenerateText(ai.BuildDescriptionPrompt(req.Title, req.Category, req.ConditionText, req.Keywords))
	if err != nil {
		log.Printf("ai generate description failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる説明生成に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text})
}
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
	text, err := h.AI.GenerateText(ai.BuildItemQAPrompt(item.Title, item.Description, item.Category, item.ConditionText, req.Question))
	if err != nil {
		log.Printf("ai item qa failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる回答生成に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text})
}

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
func (h *Handler) ListPrivateMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/private-messages"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}
	msgs, err := h.Messages.ListPrivateByItem(r.Context(), itemID, userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DM一覧の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, msgs)
}
func (h *Handler) CreatePrivateMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/private-messages"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}
	var req models.CreatePrivateMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		httpx.WriteError(w, http.StatusBadRequest, "DM本文を入力してください")
		return
	}
	msg, err := h.Messages.CreatePrivate(r.Context(), itemID, userID, req.ReceiverID, req.ParentMessageID, body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, msg)
}

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListNotifications(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "通知の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	trimmed := strings.TrimSuffix(r.URL.Path, "/read")
	id, ok := parseIDFromPath(strings.TrimPrefix(trimmed, "/api/me/notifications/"), "")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "通知IDが正しくありません")
		return
	}
	n, err := h.Users.MarkNotificationRead(r.Context(), userID, id)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "通知の確認に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, n)
}

func (h *Handler) ListSavedSearches(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListSavedSearches(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "保存検索の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}
func (h *Handler) SaveSearch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.SaveSearchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	s, err := h.Users.SaveSearch(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s)
}
func (h *Handler) DeleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	id, ok := parseIDFromPath(strings.TrimPrefix(r.URL.Path, "/api/me/saved-searches/"), "")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "保存検索IDが正しくありません")
		return
	}
	if err := h.Users.DeleteSavedSearch(r.Context(), userID, id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "保存検索の削除に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.BlockUserRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	if err := h.Users.BlockUser(r.Context(), userID, req.UserID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (h *Handler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListBlockedUsers(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ブロック一覧の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	id, ok := parseIDFromPath(strings.TrimPrefix(r.URL.Path, "/api/me/blocks/"), "")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "ユーザーIDが正しくありません")
		return
	}
	if err := h.Users.UnblockUser(r.Context(), userID, id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ブロック解除に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func (h *Handler) ListSupportMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListSupportMessages(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "運営DM履歴の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) SendSupportMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.SupportMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	subject := strings.TrimSpace(req.Subject)
	body := strings.TrimSpace(req.Body)
	if body == "" {
		httpx.WriteError(w, http.StatusBadRequest, "本文を入力してください")
		return
	}
	msg, err := h.Users.SendSupportMessage(r.Context(), userID, subject, body)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "運営への連絡に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, msg)
}
func (h *Handler) Recommend(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	items, err := h.Items.Recommend(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "おすすめ取得に失敗しました")
		return
	}
	reason := "同じC2Cマーケットでの閲覧・いいね・購入に近いシグナルを想定し、チェックリスト数、新着度、価格帯をもとに提示しています。"
	if len(items) > 0 {
		var b strings.Builder
		for _, it := range items {
			fmt.Fprintf(&b, "- %s / %s / %d円\n", it.Title, it.Category, it.PriceYen)
		}
		if user, err := h.Users.FindByID(r.Context(), userID); err == nil {
			if text, err := h.AI.GenerateText(ai.BuildRecommendationPrompt(user.Name, b.String())); err == nil {
				reason = text
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, models.RecommendationResponse{Reason: reason, Items: items})
}

func categoryReviewHints(category string) []string {
	// MerRecのようなC2C取引データで本格学習したモデルが未配置の場合でも、
	// カテゴリごとに購入者がレビューで気にしやすい観点を返します。
	// ml/merrec_recommender.py で作成したJSONモデルを将来読み込む場合も、
	// この関数を差し替えればアプリ本体のUIは変更せずに済みます。
	c := strings.TrimSpace(category)
	switch {
	case strings.Contains(c, "本") || strings.Contains(c, "教材"):
		return []string{"版・年度が古くないか", "書き込みや折れ、付属解答の有無", "初学者向けか演習量が十分か"}
	case strings.Contains(c, "ガジェット") || strings.Contains(c, "スマホ") || strings.Contains(c, "家電"):
		return []string{"バッテリー劣化や動作確認", "対応端子・OS・付属品", "保証や初期化済みか"}
	case strings.Contains(c, "音楽") || strings.Contains(c, "楽器"):
		return []string{"動作確認、音出し確認", "傷・反り・消耗部品", "付属ケースやケーブルの有無"}
	case strings.Contains(c, "食品"):
		return []string{"賞味期限・保存状態", "未開封かどうか", "アレルギー表示や受け渡しタイミング"}
	case strings.Contains(c, "ファッション"):
		return []string{"サイズ感、実寸", "汚れ・ほつれ・着用回数", "色味が写真と近いか"}
	default:
		return []string{"実物写真が十分か", "傷・汚れ・欠品の有無", "受け渡し方法と発送までの日数"}
	}
}

func splitBullets(text string, section string) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	active := false
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line == "" {
			continue
		}
		if strings.Contains(line, section) {
			active = true
			continue
		}
		if active && (strings.Contains(line, "不安点") || strings.Contains(line, "質問候補") || strings.Contains(line, "不整合")) {
			break
		}
		if active {
			out = append(out, line)
		}
	}
	if len(out) > 3 {
		return out[:3]
	}
	return out
}

func heuristicItemAnalysis(item models.Item, priceInsight string) models.ItemAIAnalysisResponse {
	risks := []string{}
	questions := []string{}
	inconsistencies := []string{}
	text := strings.ToLower(item.Title + " " + item.Description + " " + item.ConditionText + " " + item.Tags)
	if strings.Contains(item.ConditionText, "傷") || strings.Contains(item.ConditionText, "汚れ") || strings.Contains(text, "傷") || strings.Contains(text, "汚れ") {
		risks = append(risks, "傷や汚れの程度が購入判断に影響しそうです。")
		questions = append(questions, "傷や汚れが分かる写真を追加できますか？")
	}
	if item.ImageURL == "" {
		risks = append(risks, "商品画像がないため、実物状態を確認しにくいです。")
		questions = append(questions, "実物写真を追加してもらえますか？")
	}
	if item.Size == "" {
		questions = append(questions, "サイズ感や実寸を教えてもらえますか？")
	}
	if item.ShippingDays >= 7 {
		risks = append(risks, "発送までにやや時間がかかる可能性があります。")
	}
	if strings.Contains(item.Category, "本") && (strings.Contains(text, "服") || strings.Contains(text, "靴")) {
		inconsistencies = append(inconsistencies, "カテゴリは本・教材ですが、説明に衣類らしい語が含まれています。")
	}
	if strings.Contains(item.Category, "食品") && !strings.Contains(text, "賞味") && !strings.Contains(text, "期限") {
		questions = append(questions, "賞味期限や保存状態を教えてもらえますか？")
	}
	if len(risks) == 0 {
		risks = append(risks, "大きな不安点は見当たりませんが、実物状態と受け渡し条件を確認すると安心です。")
	}
	if len(questions) == 0 {
		questions = append(questions, "購入前に、状態・付属品・受け渡し方法について確認できますか？")
	}
	if len(inconsistencies) == 0 {
		inconsistencies = append(inconsistencies, "大きな不整合は見当たりません。")
	}
	return models.ItemAIAnalysisResponse{RiskPoints: risks, SuggestedQuestions: questions, Inconsistencies: inconsistencies, PriceInsight: priceInsight, CategoryReviewHints: categoryReviewHints(item.Category)}
}

func (h *Handler) CategoryKnowledge(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	httpx.WriteJSON(w, http.StatusOK, models.CategoryKnowledgeResponse{Category: category, Tips: categoryReviewHints(category)})
}

func (h *Handler) TranslateText(w http.ResponseWriter, r *http.Request) {
	var req models.AITranslateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: ""})
		return
	}
	translated, err := h.AI.GenerateText(ai.BuildTranslatePrompt(text))
	if err != nil {
		log.Printf("ai translate failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "翻訳に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: translated})
}

func (h *Handler) AnalyzeItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseIDFromPath(strings.TrimSuffix(r.URL.Path, "/analysis"), "/api/items/")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "商品IDが正しくありません")
		return
	}
	item, err := h.Items.FindByID(r.Context(), itemID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "商品が見つかりません")
		return
	}
	count, minPrice, maxPrice, avgPrice, _ := h.Items.SimilarPriceStats(r.Context(), item.Category, item.ID)
	priceInsight := "同カテゴリの比較対象が少ないため、価格妥当性は商品状態と付属品で確認してください。"
	if count > 0 {
		if float64(item.PriceYen) > avgPrice*1.25 {
			priceInsight = fmt.Sprintf("同カテゴリ%d件の平均価格は約%d円です。現在価格はやや高めなので、状態・付属品・希少性を確認すると安心です。", count, int(avgPrice))
		} else if float64(item.PriceYen) < avgPrice*0.75 {
			priceInsight = fmt.Sprintf("同カテゴリ%d件の平均価格は約%d円です。現在価格は低めなので、状態や欠品の有無を確認すると安心です。", count, int(avgPrice))
		} else {
			priceInsight = fmt.Sprintf("同カテゴリ%d件の価格帯は%d〜%d円、平均は約%d円です。現在価格は大きく外れていません。", count, minPrice, maxPrice, int(avgPrice))
		}
	}
	analysis := heuristicItemAnalysis(item, priceInsight)
	prompt := ai.BuildItemAnalysisPrompt(item.Title, item.Description, item.Category, item.ConditionText, item.PriceYen, priceInsight, strings.Join(analysis.CategoryReviewHints, " / "))
	if text, err := h.AI.GenerateText(prompt); err == nil {
		if v := splitBullets(text, "不安点"); len(v) > 0 {
			analysis.RiskPoints = v
		}
		if v := splitBullets(text, "質問候補"); len(v) > 0 {
			analysis.SuggestedQuestions = v
		}
		if v := splitBullets(text, "不整合"); len(v) > 0 {
			analysis.Inconsistencies = v
		}
	} else {
		log.Printf("ai item analysis fallback used: %v", err)
	}
	httpx.WriteJSON(w, http.StatusOK, analysis)
}

func parseIDFromPath(path string, prefix string) (int64, bool) {
	raw := path
	if prefix != "" {
		raw = strings.TrimPrefix(path, prefix)
		if raw == path {
			return 0, false
		}
	}
	raw = strings.Trim(raw, "/")
	if raw == "" || strings.Contains(raw, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
