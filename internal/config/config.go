package config

import (
	"fmt"
	"os"
)

// Config は、アプリ全体で共有する設定値をまとめる構造体です。
// 環境変数を直接いろいろな場所で読むと、どの設定が必要なのか分かりにくくなるため、
// 起動時に一度だけ読み取り、この構造体に集約します。
type Config struct {
	// Port はHTTPサーバが待ち受けるポート番号です。
	// Cloud Runでは PORT という環境変数が自動的に渡されます。
	Port string

	// FrontendOrigin はCORSで許可するフロントエンドのURLです。
	// ローカルでは http://localhost:5173、デプロイ後はVercelやGCPのURLを入れます。
	FrontendOrigin string

	// JWTSecret はログイン済みユーザーを識別するJWTの署名鍵です。
	// 漏れると他人になりすませるため、本番ではSecret Managerで管理します。
	JWTSecret string

	// MySQL接続情報です。
	MySQLUser     string
	MySQLPassword string
	MySQLHost     string
	MySQLDatabase string

	// Gemini API設定です。
	GeminiAPIKey string
	GeminiModel  string
}

// Load は環境変数からConfigを作る関数です。
// 必須値が足りない場合はエラーにして、起動時に問題を発見できるようにします。
func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		MySQLUser:      os.Getenv("MYSQL_USER"),
		MySQLPassword:  os.Getenv("MYSQL_PASSWORD"),
		MySQLHost:      os.Getenv("MYSQL_HOST"),
		MySQLDatabase:  os.Getenv("MYSQL_DATABASE"),
		GeminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		GeminiModel:    getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
	}

	// JWT_SECRETは認証の安全性に直結するため必須にしています。
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	// DB接続に必要な値も必須にしています。
	if cfg.MySQLUser == "" || cfg.MySQLPassword == "" || cfg.MySQLHost == "" || cfg.MySQLDatabase == "" {
		return Config{}, fmt.Errorf("MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_DATABASE are required")
	}

	// GeminiはAI機能に必要です。未設定でもアプリ全体は動かせますが、
	// ハッカソン必須要件に関わるため、起動時に明示的に警告できるようエラーにしています。
	if cfg.GeminiAPIKey == "" {
		return Config{}, fmt.Errorf("GEMINI_API_KEY is required")
	}

	return cfg, nil
}

// getEnv は、環境変数が設定されていればその値を返し、空ならデフォルト値を返す補助関数です。
// PORTなど、ローカル開発では省略してもよい設定に使います。
func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
