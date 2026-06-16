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
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/items", h.ListItems)
	mux.HandleFunc("POST /api/ai/generate-description", h.GenerateDescription)
	mux.HandleFunc("POST /api/ai/translate", h.TranslateText)
	mux.HandleFunc("GET /api/ai/category-knowledge", h.CategoryKnowledge)
	mux.HandleFunc("POST /api/ai/parse-search", h.ParseNaturalSearch)

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
	mux.Handle("POST /api/items", auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateItem)))

	mux.HandleFunc("/api/me/notifications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/read") {
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.ReadNotification)).ServeHTTP(w, r)
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
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/ask"):
			h.AskItem(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/messages"):
			h.ListMessages(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/messages"):
			auth.Middleware(cfg.JWTSecret, http.HandlerFunc(h.CreateMessage)).ServeHTTP(w, r)
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
