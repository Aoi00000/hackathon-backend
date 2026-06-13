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
	if viewer := h.optionalUserID(r); viewer != nil && item.SellerID != *viewer {
		blocked, _ := h.Users.AreBlocked(r.Context(), *viewer, item.SellerID)
		if blocked {
			httpx.WriteError(w, http.StatusForbidden, "ブロック関係にあるため商品を表示できません")
			return
		}
	}
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
	msg, err := h.Messages.CreatePrivate(r.Context(), itemID, userID, req.ReceiverID, body)
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
	body := strings.TrimSpace(req.Body)
	if body == "" {
		httpx.WriteError(w, http.StatusBadRequest, "本文を入力してください")
		return
	}
	msg, err := h.Users.SendSupportMessage(r.Context(), userID, body)
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
	reason := "チェックリスト数や新着順をもとに、現在購入しやすい商品を優先して提示しています。"
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
