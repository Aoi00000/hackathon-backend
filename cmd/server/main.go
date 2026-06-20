// ============================================================
// ファイル概要: hackathon-backend/cmd/server/main.go
// 役割: HTTPサーバーの起動、CORS設定、ルーティング登録、AI販売改善通知の定期実行をまとめるエントリーポイントです。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
// Package main は、AI Flea Market のGoバックエンドを起動するエントリポイントです。
//
// ここでは設定読み込み、DB接続、HTTPルーティング、認証ミドルウェアの接続を行います。
// 実際の業務ロジックは handler / repository / ai パッケージへ分離し、
// main はアプリ全体の配線だけに集中させています。
package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"hackathon-backend/internal/auth"
	"hackathon-backend/internal/config"
	"hackathon-backend/internal/db"
	"hackathon-backend/internal/handler"
	"hackathon-backend/internal/httpx"
)

// 【詳細コメント】main は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer database.Close()

	h := handler.New(cfg, database)

	// 7日以上売れ残っているAvailable商品へ、AI販売改善提案通知を作成します。
	// Cloud Schedulerを用意しなくてもローカル/Cloud Runデモで動作確認できるよう、
	// サーバ起動直後に1回、その後は24時間ごとに軽くチェックします。
	go func() {
		if created, err := h.Items.CreateStaleListingAdviceNotifications(context.Background(), 7); err != nil {
			log.Printf("stale listing advice notification failed: %v", err)
		} else if created > 0 {
			log.Printf("created %d stale listing advice notifications", created)
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if created, err := h.Items.CreateStaleListingAdviceNotifications(context.Background(), 7); err != nil {
				log.Printf("stale listing advice notification failed: %v", err)
			} else if created > 0 {
				log.Printf("created %d stale listing advice notifications", created)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/items", h.ListItems)
	mux.HandleFunc("POST /api/ai/generate-description", h.GenerateDescription)
	mux.HandleFunc("GET /api/ai/category-knowledge", h.CategoryKnowledge)
	mux.HandleFunc("POST /api/ai/parse-search", h.ParseNaturalSearch)
	mux.HandleFunc("POST /api/ai/chat", h.AIChat)

	mux.Handle("GET /api/me", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/me", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.UpdateMe)))
	mux.Handle("POST /api/me/charge", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.Charge)))
	mux.Handle("GET /api/me/items", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListMyItems)))
	mux.Handle("GET /api/me/purchases", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListPurchaseHistory)))
	mux.Handle("GET /api/me/checklist", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListChecklist)))
	mux.Handle("GET /api/me/notifications", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListNotifications)))
	mux.Handle("GET /api/me/saved-searches", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListSavedSearches)))
	mux.Handle("POST /api/me/saved-searches", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.SaveSearch)))
	mux.Handle("GET /api/me/blocks", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListBlockedUsers)))
	mux.Handle("POST /api/me/blocks", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.BlockUser)))
	mux.Handle("GET /api/me/support-messages", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListSupportMessages)))
	mux.Handle("POST /api/me/support-messages", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.SendSupportMessage)))
	mux.Handle("GET /api/me/recommendations", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.Recommend)))
	mux.Handle("GET /api/me/monthly-money-summary", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListMonthlyMoneySummary)))
	mux.Handle("GET /api/me/payment-methods", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListPaymentMethods)))
	mux.Handle("POST /api/me/payment-methods", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreatePaymentMethod)))
	mux.Handle("GET /api/me/ai-chat-threads", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListAIChatThreads)))
	mux.Handle("POST /api/me/ai-chat-threads", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateAIChatThread)))
	mux.Handle("POST /api/items", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateItem)))

	mux.HandleFunc("/api/me/notifications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/read") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ReadNotification)).ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/api/me/ai-chat-threads/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/messages") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListAIChatMessages)).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateAIChatMessage)).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.DeleteAIChatThread)).ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/api/me/payment-methods/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/default") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.SetDefaultPaymentMethod)).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.DeletePaymentMethod)).ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/api/me/saved-searches/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.DeleteSavedSearch)).ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	mux.HandleFunc("/api/me/blocks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.UnblockUser)).ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/api/items/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/purchase"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.PurchaseItem)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/ship"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ShipItem)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/complete"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CompleteItem)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/cancel"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CancelItem)).ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/checklist"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.GetChecklistStatus)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/checklist"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.AddChecklist)).ServeHTTP(w, r)
		case r.Method == http.MethodDelete && strings.HasSuffix(path, "/checklist"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.RemoveChecklist)).ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/analysis"):
			h.AnalyzeItem(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/negotiation-assist"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.GenerateNegotiationAssist)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/ask"):
			h.AskItem(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/messages"):
			h.ListMessages(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/messages"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateMessage)).ServeHTTP(w, r)
		case r.Method == http.MethodDelete && strings.Contains(path, "/messages/"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.DeleteMessage)).ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/private-messages"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListPrivateMessages)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/private-messages"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreatePrivateMessage)).ServeHTTP(w, r)
		case r.Method == http.MethodPut:
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.UpdateItem)).ServeHTTP(w, r)
		case r.Method == http.MethodGet:
			h.GetItem(w, r)
		default:
			httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	serverHandler := withCORS(cfg.FrontendOrigin, mux)
	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, serverHandler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// 【詳細コメント】withCORS は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func withCORS(frontendOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == frontendOrigin || strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
