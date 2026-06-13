package main

import (
	"log"
	"net/http"
	"strings"

	"hackathon-backend/internal/auth"
	"hackathon-backend/internal/config"
	"hackathon-backend/internal/db"
	"hackathon-backend/internal/handler"
	"hackathon-backend/internal/httpx"
)

// main はアプリケーションの起点です。
// 設定読み込み、DB接続、ルーティング、HTTPサーバ起動をこの順に行います。
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

	mux := http.NewServeMux()

	// ヘルスチェック用エンドポイントです。
	// Cloud Runやデプロイ確認で、まずこのAPIが200を返すかを見ると切り分けやすくなります。
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// 認証不要API。
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/items", h.ListItems)
	mux.HandleFunc("POST /api/ai/generate-description", h.GenerateDescription)

	// 認証が必要なAPIはauth.Middlewareで保護します。
	mux.Handle("GET /api/me", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.Me)))
	mux.Handle("GET /api/me/items", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListMyItems)))
	mux.Handle("GET /api/me/purchases", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListPurchaseHistory)))
	mux.Handle("GET /api/me/checklist", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ListChecklist)))
	mux.Handle("POST /api/items", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateItem)))

	// /api/items/{id} 系の可変パスは、標準ライブラリで分岐します。
	mux.HandleFunc("/api/items/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 商品購入。
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/purchase") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.PurchaseItem)).ServeHTTP(w, r)
			return
		}

		// 出品キャンセル。
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/cancel") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CancelItem)).ServeHTTP(w, r)
			return
		}

		// チェックリスト状態取得。
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/checklist") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.GetChecklistStatus)).ServeHTTP(w, r)
			return
		}

		// チェックリスト追加。
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/checklist") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.AddChecklist)).ServeHTTP(w, r)
			return
		}

		// チェックリスト削除。
		if r.Method == http.MethodDelete && strings.HasSuffix(path, "/checklist") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.RemoveChecklist)).ServeHTTP(w, r)
			return
		}

		// 商品についてAIに質問。
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/ask") {
			h.AskItem(w, r)
			return
		}

		// DM一覧。
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/messages") {
			h.ListMessages(w, r)
			return
		}

		// DM送信。
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/messages") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateMessage)).ServeHTTP(w, r)
			return
		}

		// 商品情報編集。
		if r.Method == http.MethodPut {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.UpdateItem)).ServeHTTP(w, r)
			return
		}

		// 商品詳細。
		if r.Method == http.MethodGet {
			h.GetItem(w, r)
			return
		}

		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	// CORSを適用したhandlerを作ります。
	serverHandler := withCORS(cfg.FrontendOrigin, mux)

	log.Printf("server listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, serverHandler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// withCORS はフロントエンドからAPIを呼べるようにするミドルウェアです。
// ハッカソンではVercel/GCPに分かれてデプロイする想定のため、CORS設定が必要です。
func withCORS(frontendOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// ローカル開発と本番フロントエンドURLを許可します。
		// 本番では "*" ではなく、実際のフロントエンドURLだけを許可する方が安全です。
		if origin == frontendOrigin || strings.HasPrefix(origin, "http://localhost:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		// ブラウザが本リクエスト前に送るプリフライトリクエストです。
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
