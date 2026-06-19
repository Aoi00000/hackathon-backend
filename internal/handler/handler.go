// Package handler は、HTTPリクエストを受け取り、入力検証、Repository呼び出し、レスポンス生成を行います。
//
// このファイルには、ハッカソンの主要機能である認証、商品、購入、コメント、通知、AI機能、自然言語検索が集約されています。
// 大きいファイルではありますが、DB操作は repository へ、外部AI呼び出しは ai へ分離し、
// Handler は「HTTP APIとして何を受け取り何を返すか」を中心に記述しています。
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
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
	// 外部AIが利用枠不足や一時混雑で失敗しても、出品作業を止めないようにします。
	// GenerateTextWithFallback は Gemini / Vertex AI が成功すればその結果を使い、
	// 失敗時は商品情報からローカルの説明文を作って返します。
	text, notice, usedFallback, err := h.AI.GenerateTextWithFallback(
		ai.BuildDescriptionPrompt(req.Title, req.Category, req.ConditionText, req.Keywords),
		func() string { return ai.FallbackDescription(req.Title, req.Category, req.ConditionText, req.Keywords) },
	)
	if err != nil {
		log.Printf("ai generate description failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる説明生成に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text, Notice: notice, UsedFallback: usedFallback})
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
	// 外部AIが利用枠不足や一時混雑で失敗しても、購入相談の体験を止めないようにします。
	// 商品説明に基づくローカル回答へフォールバックするため、デモ時にも安定して動きます。
	text, notice, usedFallback, err := h.AI.GenerateTextWithFallback(
		ai.BuildItemQAPrompt(item.Title, item.Description, item.Category, item.ConditionText, req.Question),
		func() string {
			return ai.FallbackItemQA(item.Title, item.Description, item.Category, item.ConditionText, req.Question)
		},
	)
	if err != nil {
		log.Printf("ai item qa failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AIによる回答生成に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text, Notice: notice, UsedFallback: usedFallback})
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

// ParseNaturalSearch は、商品一覧トップの「生成AIを活用した自然言語検索」を処理します。
// 役割は、ユーザーが普段の言葉で入力した検索意図を、既存の商品検索フォームと同じパラメータへ変換することです。
// Gemini / Vertex AI が使える場合は外部AIで柔軟に解釈し、429や認証未設定のときはローカル規則で最低限動かします。
func (h *Handler) ParseNaturalSearch(w http.ResponseWriter, r *http.Request) {
	var req models.NaturalSearchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		httpx.WriteError(w, http.StatusBadRequest, "自然言語検索の文章を入力してください")
		return
	}

	// まずローカル規則で解釈しておきます。
	// これにより、外部AIが混雑している場合でも検索機能としては必ず使えます。
	fallback := parseNaturalSearchLocally(query)

	// 外部AIには、既存のプルダウン候補と検索APIの項目名を明示し、JSONだけを返すように指示します。
	// 返ってきたJSONが壊れている場合も、画面を止めずにfallbackを返します。
	prompt := buildNaturalSearchPrompt(query)
	text, err := h.AI.GenerateText(prompt)
	if err != nil {
		log.Printf("natural language search fallback used: %v", err)
		fallback.Notice = "※外部AIが一時的に利用できないため、ローカル規則で検索条件を作成しました。"
		fallback.UsedFallback = true
		httpx.WriteJSON(w, http.StatusOK, fallback)
		return
	}

	parsed, err := parseNaturalSearchJSON(text)
	if err != nil {
		log.Printf("natural language search json parse fallback used: %v; raw=%s", err, text)
		fallback.Notice = "※AI応答のJSON解釈に失敗したため、ローカル規則で検索条件を作成しました。"
		fallback.UsedFallback = true
		httpx.WriteJSON(w, http.StatusOK, fallback)
		return
	}

	parsed = normalizeNaturalSearchResponse(parsed)
	if parsed.Sort == "" {
		parsed.Sort = fallback.Sort
	}
	if parsed.Explanation == "" {
		parsed.Explanation = "自然言語から検索条件を作成しました。"
	}
	httpx.WriteJSON(w, http.StatusOK, parsed)
}

// buildNaturalSearchPrompt は、自然言語検索の入力を検索パラメータJSONに変換するためのプロンプトを作ります。
// フロントエンドのプルダウン候補と完全に対応する値だけを使わせることで、AIの出力揺れを抑えます。
func buildNaturalSearchPrompt(query string) string {
	return fmt.Sprintf(`あなたは日本語フリマアプリの商品検索アシスタントです。
ユーザーの自然言語検索を、既存の商品検索フォームに入れるJSONへ変換してください。

必ず次のJSONだけを返してください。説明文、Markdown、コードブロックは禁止です。
空欄にしたい項目は空文字にしてください。

使用できるカテゴリ:
ファッション, 本・教材, ガジェット・家電, スマホ・PC周辺機器, 家具・インテリア, 日用品・生活雑貨, 美容・コスメ, スポーツ・アウトドア, ゲーム・ホビー, 音楽・楽器, チケット, ハンドメイド, 食品・飲料, その他

使用できる状態:
新品・未使用, 未使用に近い, 目立った傷や汚れなし, やや傷や汚れあり, 傷や汚れあり, 全体的に状態が悪い

使用できるsort:
recommended, new, price_asc, price_desc, checklist_desc

使用できるdeliveryWithin:
today, tomorrow, 3days, week, later

JSON形式:
{
  "q": "検索キーワード",
  "category": "カテゴリ。複数ならカンマ区切り",
  "condition": "状態。複数ならカンマ区切り",
  "status": "available か sold。通常は available",
  "minPrice": "最低価格。数字だけ",
  "maxPrice": "最高価格。数字だけ",
  "tag": "タグ検索語",
  "deliveryWithin": "発送までの日数条件",
  "sort": "並び替え",
  "explanation": "検索条件に変換した理由を日本語で60字以内"
}

例:
入力: 参考書 300円 ~ 1500円
出力: {"q":"参考書","category":"本・教材","condition":"","status":"available","minPrice":"300","maxPrice":"1500","tag":"","deliveryWithin":"","sort":"recommended","explanation":"参考書という商品種別と価格帯を検索条件にしました。"}

ユーザー入力:
%s`, query)
}

// parseNaturalSearchJSON は、Gemini / Vertex AI から返ったテキストからJSON部分を取り出して構造体にします。
// AIが誤って```json ... ```を付ける場合があるため、最初の{から最後の}までを抽出してからdecodeします。
func parseNaturalSearchJSON(text string) (models.NaturalSearchResponse, error) {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return models.NaturalSearchResponse{}, fmt.Errorf("json object not found")
	}
	jsonText := text[start : end+1]
	var result models.NaturalSearchResponse
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return models.NaturalSearchResponse{}, err
	}
	return result, nil
}

// parseNaturalSearchLocally は、外部AIが使えないときの簡易自然言語検索です。
// 完全な自然言語理解ではありませんが、デモでよく使う「予算」「きれい」「安い順」などを確実に拾います。
func parseNaturalSearchLocally(query string) models.NaturalSearchResponse {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)
	res := models.NaturalSearchResponse{Status: "available", Sort: "recommended"}

	// 価格条件を抽出します。
	// 例: 「1万円以内」では maxPrice=10000、
	//     「500円以上」では minPrice=500、
	//     「参考書 300円 ~ 1500円」では minPrice=300 / maxPrice=1500 にします。
	if min, max := extractPriceRangeFromJapanese(q); min > 0 || max > 0 {
		if min > 0 {
			res.MinPrice = strconv.Itoa(min)
		}
		if max > 0 {
			res.MaxPrice = strconv.Itoa(max)
		}
	} else {
		if max := extractMaxPriceFromJapanese(q); max > 0 {
			res.MaxPrice = strconv.Itoa(max)
		}
		if min := extractMinPriceFromJapanese(q); min > 0 {
			res.MinPrice = strconv.Itoa(min)
		}
	}

	// 並び替えを抽出します。
	if strings.Contains(q, "安い順") || strings.Contains(q, "安く") || strings.Contains(q, "安いもの") {
		res.Sort = "price_asc"
	}
	if strings.Contains(q, "高い順") || strings.Contains(q, "高いもの") {
		res.Sort = "price_desc"
	}
	if strings.Contains(q, "新しい") || strings.Contains(q, "新着") {
		res.Sort = "new"
	}
	if strings.Contains(q, "人気") || strings.Contains(q, "チェックリスト") || strings.Contains(q, "いいね") {
		res.Sort = "checklist_desc"
	}

	// 商品状態を抽出します。
	if strings.Contains(q, "新品") || strings.Contains(q, "未使用") {
		res.Condition = "新品・未使用,未使用に近い"
	}
	if strings.Contains(q, "きれい") || strings.Contains(q, "綺麗") || strings.Contains(q, "使用感が少") || strings.Contains(q, "美品") {
		res.Condition = joinNonEmpty(res.Condition, "未使用に近い,目立った傷や汚れなし")
	}
	if strings.Contains(q, "傷") || strings.Contains(q, "汚れ") {
		res.Condition = joinNonEmpty(res.Condition, "やや傷や汚れあり,傷や汚れあり")
	}

	// カテゴリを抽出します。
	categories := []string{}
	addCategory := func(category string) {
		if !containsString(categories, category) {
			categories = append(categories, category)
		}
	}
	switch {
	case strings.Contains(q, "参考書") || strings.Contains(q, "教科書") || strings.Contains(q, "本") || strings.Contains(q, "教材") || strings.Contains(q, "数学") || strings.Contains(q, "英語"):
		addCategory("本・教材")
	case strings.Contains(q, "スマホ") || strings.Contains(q, "pc") || strings.Contains(lower, "usb") || strings.Contains(q, "イヤホン") || strings.Contains(q, "充電"):
		addCategory("スマホ・PC周辺機器")
	case strings.Contains(q, "家電") || strings.Contains(q, "ライト") || strings.Contains(q, "ガジェット"):
		addCategory("ガジェット・家電")
	case strings.Contains(q, "服") || strings.Contains(q, "パーカー") || strings.Contains(q, "シャツ") || strings.Contains(q, "ファッション"):
		addCategory("ファッション")
	case strings.Contains(q, "食品") || strings.Contains(q, "食べ物") || strings.Contains(q, "スープ") || strings.Contains(q, "玉ねぎ") || strings.Contains(q, "たまねぎ"):
		addCategory("食品・飲料")
	case strings.Contains(q, "家具") || strings.Contains(q, "インテリア") || strings.Contains(q, "植物"):
		addCategory("家具・インテリア")
	case strings.Contains(q, "ゲーム") || strings.Contains(q, "カード") || strings.Contains(q, "ホビー"):
		addCategory("ゲーム・ホビー")
	case strings.Contains(q, "音楽") || strings.Contains(q, "楽器") || strings.Contains(q, "ギター"):
		addCategory("音楽・楽器")
	}
	res.Category = strings.Join(categories, ",")

	// 発送までの日数を抽出します。
	switch {
	case strings.Contains(q, "今日") || strings.Contains(q, "本日"):
		res.DeliveryWithin = "today"
	case strings.Contains(q, "明日"):
		res.DeliveryWithin = "tomorrow"
	case strings.Contains(q, "3日") || strings.Contains(q, "三日"):
		res.DeliveryWithin = "3days"
	case strings.Contains(q, "1週間") || strings.Contains(q, "一週間"):
		res.DeliveryWithin = "week"
	}

	// 余った語はキーワードとして利用します。カテゴリや価格語だけで絞れる場合は空でも構いません。
	res.Q = cleanupNaturalSearchKeyword(q)
	if res.Category != "" || res.Condition != "" || res.MaxPrice != "" || res.MinPrice != "" || res.Sort != "recommended" {
		res.Explanation = "自然言語から価格・カテゴリ・状態・並び順を推定しました。"
	} else {
		res.Explanation = "入力文をキーワード検索として利用します。"
	}
	return normalizeNaturalSearchResponse(res)
}

func normalizeNaturalSearchResponse(res models.NaturalSearchResponse) models.NaturalSearchResponse {
	res.Q = strings.TrimSpace(res.Q)
	res.Category = strings.Trim(strings.TrimSpace(res.Category), ",")
	res.Condition = strings.Trim(strings.TrimSpace(res.Condition), ",")
	res.Status = strings.Trim(strings.TrimSpace(res.Status), ",")
	res.MinPrice = digitsOnly(res.MinPrice)
	res.MaxPrice = digitsOnly(res.MaxPrice)
	res.Tag = strings.TrimSpace(res.Tag)
	res.DeliveryWithin = strings.TrimSpace(res.DeliveryWithin)
	res.Sort = strings.TrimSpace(res.Sort)
	if res.Sort == "" {
		res.Sort = "recommended"
	}
	return res
}

func extractPriceRangeFromJapanese(text string) (int, int) {
	// 日本語の自然言語検索では、「300円 ~ 1500円」「300円〜1500円」
	// 「300-1500円」のように、範囲指定がさまざまな表記で入力されます。
	// この関数では、範囲表記だけを先に拾い、最小価格と最大価格に分解します。
	// 範囲が取れなかった場合は 0, 0 を返し、既存の「以内」「以上」処理に任せます。
	patterns := []string{
		`([0-9０-９,]+)\s*円?\s*(?:~|〜|－|-|から)\s*([0-9０-９,]+)\s*円`,
		`([0-9０-９,]+)\s*円\s*(?:~|〜|－|-|から)\s*([0-9０-９,]+)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		m := re.FindStringSubmatch(text)
		if len(m) >= 3 {
			left := japaneseNumberToInt(m[1])
			right := japaneseNumberToInt(m[2])
			if left > 0 && right > 0 {
				if left <= right {
					return left, right
				}
				return right, left
			}
		}
	}
	return 0, 0
}

func extractMaxPriceFromJapanese(text string) int {
	patterns := []string{`([0-9０-９]+)\s*万\s*円?\s*(以内|以下|まで|未満)?`, `([0-9０-９,]+)\s*円\s*(以内|以下|まで|未満)`}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		m := re.FindStringSubmatch(text)
		if len(m) >= 2 {
			value := japaneseNumberToInt(m[1])
			if strings.Contains(pattern, "万") {
				value *= 10000
			}
			return value
		}
	}
	return 0
}

func extractMinPriceFromJapanese(text string) int {
	re := regexp.MustCompile(`([0-9０-９,]+)\s*円\s*(以上|から)`)
	m := re.FindStringSubmatch(text)
	if len(m) >= 2 {
		return japaneseNumberToInt(m[1])
	}
	return 0
}

func japaneseNumberToInt(text string) int {
	text = strings.NewReplacer("０", "0", "１", "1", "２", "2", "３", "3", "４", "4", "５", "5", "６", "6", "７", "7", "８", "8", "９", "9", ",", "").Replace(text)
	value, _ := strconv.Atoi(digitsOnly(text))
	return value
}

func digitsOnly(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func joinNonEmpty(existing, add string) string {
	parts := []string{}
	for _, value := range strings.Split(existing+","+add, ",") {
		value = strings.TrimSpace(value)
		if value != "" && !containsString(parts, value) {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ",")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cleanupNaturalSearchKeyword(text string) string {
	replacers := []string{"予算", "以内", "以下", "まで", "安い順", "高い順", "並べて", "探して", "検索", "使用感が少なくて", "使用感が少ない", "きれいな", "綺麗な", "もの", "商品", "ください", "して", "で", "を", "に", "が", "の", "~", "〜", "－", "-"}
	cleaned := text
	for _, word := range replacers {
		cleaned = strings.ReplaceAll(cleaned, word, " ")
	}
	cleaned = regexp.MustCompile(`[0-9０-９,]+\s*(万円|万|円)`).ReplaceAllString(cleaned, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "で" || cleaned == "を" || cleaned == "に" || cleaned == "の" || cleaned == "が" {
		return ""
	}
	if len([]rune(cleaned)) > 24 {
		cleaned = string([]rune(cleaned)[:24])
	}
	return cleaned
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

func (h *Handler) ListMonthlyMoneySummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListMonthlyMoneySummary(r.Context(), userID, 6)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "月別収支の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	data, err := h.Users.ListPaymentMethods(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "支払い方法の取得に失敗しました")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) CreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	var req models.CreatePaymentMethodRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	method, err := h.Users.CreatePaymentMethod(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, method)
}

func (h *Handler) SetDefaultPaymentMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	trimmed := strings.TrimSuffix(r.URL.Path, "/default")
	id, ok := parseIDFromPath(strings.TrimPrefix(trimmed, "/api/me/payment-methods/"), "")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "支払い方法IDが正しくありません")
		return
	}
	if err := h.Users.SetDefaultPaymentMethod(r.Context(), userID, id); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	id, ok := parseIDFromPath(strings.TrimPrefix(r.URL.Path, "/api/me/payment-methods/"), "")
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "支払い方法IDが正しくありません")
		return
	}
	if err := h.Users.DeletePaymentMethod(r.Context(), userID, id); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseItemMessageIDs(path string) (int64, int64, bool) {
	trimmed := strings.TrimPrefix(path, "/api/items/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 || parts[1] != "messages" {
		return 0, 0, false
	}
	itemID, err1 := strconv.ParseInt(parts[0], 10, 64)
	messageID, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || itemID <= 0 || messageID <= 0 {
		return 0, 0, false
	}
	return itemID, messageID, true
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "ログインが必要です")
		return
	}
	itemID, messageID, ok := parseItemMessageIDs(r.URL.Path)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "コメントIDが正しくありません")
		return
	}
	if err := h.Messages.DeletePublicBySeller(r.Context(), itemID, messageID, userID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) AIChat(w http.ResponseWriter, r *http.Request) {
	var req models.AIChatRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "JSONの形式が正しくありません")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		httpx.WriteError(w, http.StatusBadRequest, "質問を入力してください")
		return
	}
	text, notice, usedFallback, err := h.AI.GenerateTextWithFallback(
		ai.BuildGeneralChatPrompt(message),
		func() string { return ai.FallbackGeneralChat(message) },
	)
	if err != nil {
		log.Printf("ai chat failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "AI対話に失敗しました: "+err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, models.AITextResponse{Text: text, Notice: notice, UsedFallback: usedFallback})
}
