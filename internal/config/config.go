// ============================================================
// ファイル概要: hackathon-backend/internal/config/config.go
// 役割: 環境変数からDB接続情報、JWT秘密鍵、AI関連設定を読み込みます。
//
// 読み方の目安:
// 1. まずpackage/importを確認し、このファイルがどの層に属するかを把握します。
// 2. type定義では、DB/API/画面で受け渡すデータの形を確認します。
// 3. func定義では、入力検証、DB処理、AI呼び出し、レスポンス整形の順に読むと流れを追いやすくなります。
//
// ============================================================
package config

import (
	"fmt"
	"os"
)

// Config は、アプリ全体で共有する設定値をまとめる構造体です。
// 環境変数を直接いろいろな場所で読むと、どの設定が必要なのか分かりにくくなるため、
// 起動時に一度だけ読み取り、この構造体に集約します。
// 【詳細コメント】Config は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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
// 【詳細コメント】Load は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
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

// 【詳細コメント】getEnv は、この層の責務を小さく保つための宣言です。入力・出力・DB/APIとの対応を意識して読むと、全体の流れを追いやすくなります。
func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
