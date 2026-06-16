package config

import (
	"fmt"
	"os"
)

// Config は、アプリ全体で共有する設定値をまとめる構造体です。
// 環境変数を直接いろいろな場所で読むと、どの設定が必要なのか分かりにくくなるため、
// 起動時に一度だけ読み取り、この構造体に集約します。
type Config struct {
	Port           string
	FrontendOrigin string
	JWTSecret      string

	MySQLUser     string
	MySQLPassword string
	MySQLHost     string
	MySQLDatabase string

	// AIProvider は ai_studio / vertex のいずれかです。
	// ai_studio はGoogle AI StudioのAPIキーを使う簡易方式、vertex は研修資料のVertex AI方式です。
	AIProvider      string
	GeminiAPIKey    string
	GeminiModel     string
	GoogleProjectID string
	VertexLocation  string
}

// Load は環境変数からConfigを作る関数です。
func Load() (Config, error) {
	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		FrontendOrigin:  getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		MySQLUser:       os.Getenv("MYSQL_USER"),
		MySQLPassword:   os.Getenv("MYSQL_PASSWORD"),
		MySQLHost:       os.Getenv("MYSQL_HOST"),
		MySQLDatabase:   os.Getenv("MYSQL_DATABASE"),
		AIProvider:      getEnv("AI_PROVIDER", "ai_studio"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		GeminiModel:     getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		GoogleProjectID: getEnv("GOOGLE_CLOUD_PROJECT", os.Getenv("PROJECT_ID")),
		VertexLocation:  getEnv("VERTEX_LOCATION", "global"),
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.MySQLUser == "" || cfg.MySQLPassword == "" || cfg.MySQLHost == "" || cfg.MySQLDatabase == "" {
		return Config{}, fmt.Errorf("MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_DATABASE are required")
	}
	if cfg.AIProvider != "ai_studio" && cfg.AIProvider != "vertex" {
		return Config{}, fmt.Errorf("AI_PROVIDER must be ai_studio or vertex")
	}
	// AI関連の認証情報は必須にはしません。
	// 未設定・利用枠不足・429の場合でも、AIエンドポイント側でローカルの簡易生成へ
	// フォールバックさせることで、ハッカソンのデモ画面全体が止まらないようにします。
	// 本格運用では .env に GEMINI_API_KEY または Vertex AI のPROJECT設定を入れてください。
	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
